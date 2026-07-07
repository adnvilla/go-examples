// Demonstrates the transactional outbox pattern: business state and the
// events announcing it are committed in one transaction, and a relay
// publishes them afterwards — closing the dual-write gap where a crash
// between "commit" and "publish" silently loses events. The crash window
// that remains (publish succeeded, mark didn't) is shown too: it downgrades
// the guarantee to at-least-once, which idempotent consumers absorb.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	_ "modernc.org/sqlite" // registers the "sqlite" driver with database/sql
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()      //nolint:errcheck
	db.SetMaxOpenConns(1) // :memory: gives each pool connection its own DB

	store := NewStore(db)
	if err := store.Init(ctx); err != nil {
		return err
	}
	broker := NewBroker()

	if err := dualWriteProblem(ctx, db, broker); err != nil {
		return err
	}
	if err := outboxWrite(ctx, store); err != nil {
		return err
	}
	if err := relayHappyPath(ctx, store, broker); err != nil {
		return err
	}
	return relayCrashWindow(ctx, store, broker)
}

// dualWriteProblem shows why the pattern exists: commit the state, then
// crash before telling the broker. The order exists, the event is gone, and
// nothing anywhere records that anything is missing.
func dualWriteProblem(ctx context.Context, db *sql.DB, broker *Broker) error {
	fmt.Println("--- dual write, the broken way: commit, then crash before publish ---")

	if _, err := db.ExecContext(ctx, `INSERT INTO orders (item) VALUES (?)`, "keyboard"); err != nil {
		return fmt.Errorf("inserting order: %w", err)
	}
	fmt.Println("app: order committed")
	fmt.Println("app: CRASH before broker.Publish — the event never existed")

	var orders int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM orders`).Scan(&orders); err != nil {
		return fmt.Errorf("counting orders: %w", err)
	}
	fmt.Printf("result: %d order in the database, %d events delivered -> silently inconsistent\n",
		orders, len(broker.delivered))
	return nil
}

// outboxWrite shows the fix: the event rides in the same transaction as the
// state, so a rollback removes both and a commit persists both.
func outboxWrite(ctx context.Context, store *Store) error {
	fmt.Println("\n--- outbox write: state + event in ONE transaction ---")

	orderID, err := store.PlaceOrder(ctx, "trackball")
	if err != nil {
		return err
	}
	fmt.Printf("app: order %d and its event committed atomically\n", orderID)

	pending, err := store.PendingEvents(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("outbox: %d event pending, safe on disk next to the order\n", len(pending))
	return nil
}

// relayHappyPath drains the outbox: publish, then mark dispatched.
func relayHappyPath(ctx context.Context, store *Store, broker *Broker) error {
	fmt.Println("\n--- relay: poll pending -> publish -> mark dispatched ---")
	if err := Relay(ctx, store, broker, false); err != nil {
		return err
	}

	pending, err := store.PendingEvents(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("outbox: %d events pending after relay run\n", len(pending))
	return nil
}

// relayCrashWindow shows the residual window honestly: the relay publishes,
// dies before marking, and the rerun redelivers the same event — which the
// idempotent consumer drops. At-least-once + dedupe is the contract.
func relayCrashWindow(ctx context.Context, store *Store, broker *Broker) error {
	fmt.Println("\n--- crash window: publish succeeded, mark didn't -> redelivery ---")

	orderID, err := store.PlaceOrder(ctx, "desk mat")
	if err != nil {
		return err
	}
	fmt.Printf("app: order %d and its event committed atomically\n", orderID)

	fmt.Println("relay run 1 (crashes mid-flight):")
	if err := Relay(ctx, store, broker, true); err != nil {
		return err
	}
	fmt.Println("relay run 2 (after restart):")
	if err := Relay(ctx, store, broker, false); err != nil {
		return err
	}

	pending, err := store.PendingEvents(ctx)
	if err != nil {
		return err
	}
	if len(pending) != 0 {
		return errors.New("outbox should be drained after the second relay run")
	}
	fmt.Printf("result: %d unique events delivered, %d duplicate absorbed, outbox drained\n",
		len(broker.delivered), broker.dupes)
	return nil
}
