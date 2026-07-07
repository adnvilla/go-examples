package main

import (
	"context"
	"database/sql"
	"fmt"
)

// Store owns the business tables and the outbox table. The pattern's core
// rule: state changes and the events announcing them are written in the SAME
// transaction, so they commit or vanish together.
type Store struct {
	db *sql.DB
}

// NewStore wraps an open database handle.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Init creates the schema: the business table and the outbox next to it.
func (s *Store) Init(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE orders (
			id   INTEGER PRIMARY KEY AUTOINCREMENT,
			item TEXT NOT NULL
		);
		CREATE TABLE outbox (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			topic      TEXT NOT NULL,
			payload    TEXT NOT NULL,
			dispatched INTEGER NOT NULL DEFAULT 0
		)`)
	if err != nil {
		return fmt.Errorf("creating schema: %w", err)
	}
	return nil
}

// PlaceOrder writes the order AND its event atomically. There is no broker
// call here — publishing is the relay's job, later. If this transaction
// rolls back, neither the order nor the event ever existed.
func (s *Store) PlaceOrder(ctx context.Context, item string) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("beginning tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	res, err := tx.ExecContext(ctx, `INSERT INTO orders (item) VALUES (?)`, item)
	if err != nil {
		return 0, fmt.Errorf("inserting order: %w", err)
	}
	orderID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("reading order id: %w", err)
	}

	payload := fmt.Sprintf(`{"order_id": %d, "item": %q}`, orderID, item)
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO outbox (topic, payload) VALUES (?, ?)`, "order.placed", payload); err != nil {
		return 0, fmt.Errorf("inserting outbox event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("committing: %w", err)
	}
	return orderID, nil
}

// PendingEvents returns undispatched events in insertion order.
func (s *Store) PendingEvents(ctx context.Context) ([]Event, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, topic, payload FROM outbox WHERE dispatched = 0 ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("querying outbox: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.Topic, &e.Payload); err != nil {
			return nil, fmt.Errorf("scanning event: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating outbox: %w", err)
	}
	return events, nil
}

// MarkDispatched records that an event reached the broker.
func (s *Store) MarkDispatched(ctx context.Context, eventID int64) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE outbox SET dispatched = 1 WHERE id = ?`, eventID); err != nil {
		return fmt.Errorf("marking event %d dispatched: %w", eventID, err)
	}
	return nil
}

// Event is one row of the outbox.
type Event struct {
	ID      int64
	Topic   string
	Payload string
}

// Broker stands in for Kafka/Rabbit/SNS. It remembers event IDs so the
// consumer side can demonstrate idempotency under at-least-once delivery.
type Broker struct {
	seen      map[int64]bool
	delivered []string
	dupes     int
}

// NewBroker returns an empty in-memory broker.
func NewBroker() *Broker {
	return &Broker{seen: map[int64]bool{}}
}

// Publish delivers an event; redeliveries of an already-seen event ID are
// counted and ignored — the consumer-side idempotency the pattern requires.
func (b *Broker) Publish(e Event) {
	if b.seen[e.ID] {
		b.dupes++
		fmt.Printf("  broker: duplicate event %d ignored (idempotent consumer)\n", e.ID)
		return
	}
	b.seen[e.ID] = true
	b.delivered = append(b.delivered, fmt.Sprintf("%s %s", e.Topic, e.Payload))
	fmt.Printf("  broker: delivered event %d: %s %s\n", e.ID, e.Topic, e.Payload)
}

// Relay is the poller: read pending events, publish, mark dispatched.
// crashBeforeMark simulates dying in the window between the publish and the
// mark — the window that makes the guarantee at-least-once, not exactly-once.
func Relay(ctx context.Context, store *Store, broker *Broker, crashBeforeMark bool) error {
	events, err := store.PendingEvents(ctx)
	if err != nil {
		return err
	}
	for _, e := range events {
		broker.Publish(e)
		if crashBeforeMark {
			fmt.Println("  relay: CRASH after publish, before mark — event stays pending")
			return nil
		}
		if err := store.MarkDispatched(ctx, e.ID); err != nil {
			return err
		}
	}
	return nil
}
