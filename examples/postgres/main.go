// Demonstrates PostgreSQL beyond basic queries, with pgx: connection pooling,
// $1 placeholders, and two features the database/sql surface hides — a live
// write-skew anomaly that READ COMMITTED allows and SERIALIZABLE detects
// (SQLSTATE 40001, the "retry me" error), and LISTEN/NOTIFY pub/sub straight
// from the database.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The credentials match this repo's docker-compose postgres service; real
// services read the DSN from configuration, never a source literal.
const dsn = "postgres://postgres:secret@localhost:5432/examples" //nolint:gosec // local docker-compose demo credentials

// serializationFailure is PostgreSQL's SQLSTATE for "this transaction is a
// casualty of serializable ordering — roll back and retry".
const serializationFailure = "40001"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("creating pool: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("postgres is not reachable on localhost:5432 (docker compose up -d postgres): %w", err)
	}

	if err := setupSchema(ctx, pool); err != nil {
		return err
	}
	if err := writeSkew(ctx, pool, pgx.ReadCommitted); err != nil {
		return err
	}
	if err := resetDoctors(ctx, pool); err != nil {
		return err
	}
	if err := writeSkew(ctx, pool, pgx.Serializable); err != nil {
		return err
	}
	return listenNotify(ctx, pool)
}

func setupSchema(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		DROP TABLE IF EXISTS doctors;
		CREATE TABLE doctors (
			name    TEXT PRIMARY KEY,
			on_call BOOLEAN NOT NULL
		);
		INSERT INTO doctors (name, on_call) VALUES ('alice', true), ('bob', true);`)
	if err != nil {
		return fmt.Errorf("setting up schema: %w", err)
	}
	fmt.Println("--- schema: two doctors, both on call; invariant: at least one on call ---")
	return nil
}

func resetDoctors(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `UPDATE doctors SET on_call = true`); err != nil {
		return fmt.Errorf("resetting doctors: %w", err)
	}
	return nil
}

// writeSkew runs the textbook write-skew schedule: two transactions each
// check the invariant ("someone else is still on call"), then update
// *different* rows, then commit. No row is written by both, so no lock
// blocks anything — only true serializability can catch the conflict.
func writeSkew(ctx context.Context, pool *pgxpool.Pool, level pgx.TxIsoLevel) error {
	fmt.Printf("\n--- write skew under %s ---\n", level)

	// Both alice's and bob's transactions start and read before either writes.
	txAlice, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: level})
	if err != nil {
		return fmt.Errorf("beginning alice's tx: %w", err)
	}
	defer txAlice.Rollback(ctx) //nolint:errcheck
	txBob, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: level})
	if err != nil {
		return fmt.Errorf("beginning bob's tx: %w", err)
	}
	defer txBob.Rollback(ctx) //nolint:errcheck

	countOnCall := func(tx pgx.Tx, who string) (int, error) {
		var n int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM doctors WHERE on_call`).Scan(&n); err != nil {
			return 0, fmt.Errorf("%s counting on-call doctors: %w", who, err)
		}
		fmt.Printf("%s's tx: sees %d doctors on call -> ok to leave\n", who, n)
		return n, nil
	}
	if _, err := countOnCall(txAlice, "alice"); err != nil {
		return err
	}
	if _, err := countOnCall(txBob, "bob"); err != nil {
		return err
	}

	// Each updates only its own row — disjoint writes, no lock contention.
	if _, err := txAlice.Exec(ctx, `UPDATE doctors SET on_call = false WHERE name = 'alice'`); err != nil {
		return fmt.Errorf("alice going off call: %w", err)
	}
	// Depending on the Postgres version, serializable may reject bob at this
	// UPDATE or later at COMMIT — treat both spots as the same outcome.
	_, bobErr := txBob.Exec(ctx, `UPDATE doctors SET on_call = false WHERE name = 'bob'`)
	if bobErr != nil && !isSerializationFailure(bobErr) {
		return fmt.Errorf("bob going off call: %w", bobErr)
	}

	if err := txAlice.Commit(ctx); err != nil {
		return fmt.Errorf("committing alice's tx: %w", err)
	}
	fmt.Println("alice's tx: committed")

	if bobErr == nil {
		bobErr = txBob.Commit(ctx)
	}
	switch {
	case bobErr == nil:
		fmt.Println("bob's tx: committed")
	case isSerializationFailure(bobErr):
		fmt.Println("bob's tx: rejected with SQLSTATE 40001 (serialization failure) — retrying")
		if err := retryBob(ctx, pool, level); err != nil {
			return err
		}
	default:
		return fmt.Errorf("committing bob's tx: %w", bobErr)
	}

	var onCall int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM doctors WHERE on_call`).Scan(&onCall); err != nil {
		return fmt.Errorf("checking invariant: %w", err)
	}
	verdict := "PRESERVED"
	if onCall == 0 {
		verdict = "BROKEN"
	}
	fmt.Printf("doctors on call now: %d -> invariant %s\n", onCall, verdict)
	return nil
}

// retryBob is what SQLSTATE 40001 asks for: run the whole transaction again
// from the top. This time the fresh read sees the invariant would break, so
// bob stays on call.
func retryBob(ctx context.Context, pool *pgxpool.Pool, level pgx.TxIsoLevel) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: level})
	if err != nil {
		return fmt.Errorf("beginning bob's retry: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var n int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM doctors WHERE on_call`).Scan(&n); err != nil {
		return fmt.Errorf("bob's retry counting: %w", err)
	}
	if n <= 1 {
		fmt.Printf("bob's retry: sees only %d doctor on call -> stays on call\n", n)
		return tx.Commit(ctx)
	}
	if _, err := tx.Exec(ctx, `UPDATE doctors SET on_call = false WHERE name = 'bob'`); err != nil {
		return fmt.Errorf("bob's retry going off call: %w", err)
	}
	return tx.Commit(ctx)
}

func isSerializationFailure(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == serializationFailure
}

// listenNotify shows Postgres as a lightweight pub/sub: a dedicated
// connection LISTENs on a channel, any other session NOTIFYs, and the
// payload arrives without polling.
func listenNotify(ctx context.Context, pool *pgxpool.Pool) error {
	fmt.Println("\n--- LISTEN/NOTIFY: pub/sub from the database ---")

	// LISTEN needs a dedicated connection: notifications are delivered on
	// the session that listened, so it can't go back to the pool while
	// subscribed.
	listener, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquiring listener connection: %w", err)
	}
	defer listener.Release()

	if _, err := listener.Exec(ctx, `LISTEN order_events`); err != nil {
		return fmt.Errorf("subscribing: %w", err)
	}
	fmt.Println("listener: subscribed to channel order_events")

	if _, err := pool.Exec(ctx, `SELECT pg_notify('order_events', 'order-1001 shipped')`); err != nil {
		return fmt.Errorf("notifying: %w", err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	notification, err := listener.Conn().WaitForNotification(waitCtx)
	if err != nil {
		return fmt.Errorf("waiting for notification: %w", err)
	}
	fmt.Printf("listener: received %q on channel %q\n", notification.Payload, notification.Channel)
	return nil
}
