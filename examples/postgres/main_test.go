package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// requirePostgres skips unless POSTGRES_LOCAL=1 (same convention as the
// dynamodb and distributed-lock examples) and returns a pool with a
// test-scoped doctors table.
func requirePostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if os.Getenv("POSTGRES_LOCAL") == "" {
		t.Skip("set POSTGRES_LOCAL=1 to run Postgres integration tests (requires local Postgres on :5432)")
	}

	pool, err := pgxpool.New(t.Context(), dsn)
	if err != nil {
		t.Fatalf("creating pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(t.Context()); err != nil {
		t.Fatalf("postgres not reachable: %v", err)
	}
	if err := setupSchema(t.Context(), pool); err != nil {
		t.Fatalf("schema: %v", err)
	}
	return pool
}

// TestWriteSkewAllowedUnderReadCommitted pins the anomaly itself: both
// disjoint-row transactions commit and the invariant breaks. If this test
// ever fails, the demo's premise changed.
func TestWriteSkewAllowedUnderReadCommitted(t *testing.T) {
	pool := requirePostgres(t)

	if err := writeSkew(t.Context(), pool, pgx.ReadCommitted); err != nil {
		t.Fatalf("writeSkew: %v", err)
	}
	var onCall int
	if err := pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM doctors WHERE on_call`).Scan(&onCall); err != nil {
		t.Fatalf("counting: %v", err)
	}
	if onCall != 0 {
		t.Fatalf("on-call doctors = %d, want 0 — read committed should allow the skew", onCall)
	}
}

// TestWriteSkewPreventedUnderSerializable pins the fix: one transaction is
// rejected with 40001, the retry declines, and the invariant holds.
func TestWriteSkewPreventedUnderSerializable(t *testing.T) {
	pool := requirePostgres(t)

	if err := writeSkew(t.Context(), pool, pgx.Serializable); err != nil {
		t.Fatalf("writeSkew: %v", err)
	}
	var onCall int
	if err := pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM doctors WHERE on_call`).Scan(&onCall); err != nil {
		t.Fatalf("counting: %v", err)
	}
	if onCall != 1 {
		t.Fatalf("on-call doctors = %d, want 1 — serializable should prevent the skew", onCall)
	}
}

func TestListenNotifyRoundTrip(t *testing.T) {
	pool := requirePostgres(t)

	listener, err := pool.Acquire(t.Context())
	if err != nil {
		t.Fatalf("acquiring listener: %v", err)
	}
	defer listener.Release()

	if _, err := listener.Exec(t.Context(), `LISTEN test_events`); err != nil {
		t.Fatalf("LISTEN: %v", err)
	}
	if _, err := pool.Exec(t.Context(), `SELECT pg_notify('test_events', 'ping')`); err != nil {
		t.Fatalf("NOTIFY: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	n, err := listener.Conn().WaitForNotification(ctx)
	if err != nil {
		t.Fatalf("waiting: %v", err)
	}
	if n.Channel != "test_events" || n.Payload != "ping" {
		t.Fatalf("notification = %q on %q, want \"ping\" on \"test_events\"", n.Payload, n.Channel)
	}
}
