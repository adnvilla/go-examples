package main

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	store := NewStore(db)
	if err := store.Init(t.Context()); err != nil {
		t.Fatalf("initializing schema: %v", err)
	}
	return store
}

func TestPlaceOrderWritesStateAndEventAtomically(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)

	orderID, err := store.PlaceOrder(t.Context(), "keyboard")
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if orderID == 0 {
		t.Fatal("expected a generated order id")
	}

	events, err := store.PendingEvents(t.Context())
	if err != nil {
		t.Fatalf("PendingEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("pending events = %d, want 1", len(events))
	}
	if events[0].Topic != "order.placed" {
		t.Errorf("topic = %q, want %q", events[0].Topic, "order.placed")
	}
}

func TestRelayDispatchesOnceOnHappyPath(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	broker := NewBroker()

	if _, err := store.PlaceOrder(t.Context(), "trackball"); err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if err := Relay(t.Context(), store, broker, false); err != nil {
		t.Fatalf("Relay: %v", err)
	}

	if len(broker.delivered) != 1 || broker.dupes != 0 {
		t.Fatalf("delivered=%d dupes=%d, want 1/0", len(broker.delivered), broker.dupes)
	}
	pending, err := store.PendingEvents(t.Context())
	if err != nil {
		t.Fatalf("PendingEvents: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending after relay = %d, want 0", len(pending))
	}
	// Rerunning the relay against a drained outbox must be a no-op.
	if err := Relay(t.Context(), store, broker, false); err != nil {
		t.Fatalf("second Relay: %v", err)
	}
	if len(broker.delivered) != 1 || broker.dupes != 0 {
		t.Fatalf("after rerun: delivered=%d dupes=%d, want 1/0", len(broker.delivered), broker.dupes)
	}
}

func TestCrashBetweenPublishAndMarkRedelivers(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	broker := NewBroker()

	if _, err := store.PlaceOrder(t.Context(), "desk mat"); err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}

	// First run publishes but "crashes" before marking dispatched.
	if err := Relay(t.Context(), store, broker, true); err != nil {
		t.Fatalf("crashing Relay: %v", err)
	}
	pending, err := store.PendingEvents(t.Context())
	if err != nil {
		t.Fatalf("PendingEvents: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("pending after crash = %d, want 1 — the event must survive", len(pending))
	}

	// The rerun redelivers; the consumer's dedupe absorbs it.
	if err := Relay(t.Context(), store, broker, false); err != nil {
		t.Fatalf("recovery Relay: %v", err)
	}
	if len(broker.delivered) != 1 {
		t.Errorf("unique deliveries = %d, want 1", len(broker.delivered))
	}
	if broker.dupes != 1 {
		t.Errorf("absorbed duplicates = %d, want 1", broker.dupes)
	}
}
