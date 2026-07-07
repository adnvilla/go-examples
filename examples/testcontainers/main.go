// Demonstrates integration tests that own their infrastructure: instead of
// requiring a docker-compose service and an env-var guard (the pattern used
// by this repo's dynamodb/postgres/kafka examples), each test starts its own
// throwaway PostgreSQL container with testcontainers-go, waits for readiness,
// runs against the real database, and tears everything down. The tests are
// the example; this file holds the code under test.
package main

import (
	"context"
	"database/sql"
	"fmt"
)

// NoteStore is the small repository under test — real SQL against a real
// PostgreSQL, no mocks.
type NoteStore struct {
	db *sql.DB
}

// NewNoteStore wraps an open database handle.
func NewNoteStore(db *sql.DB) *NoteStore {
	return &NoteStore{db: db}
}

// Init creates the schema. Tests call it against a container that started
// empty seconds ago — schema setup is part of the code under test, not a
// fixture hidden in CI configuration.
func (s *NoteStore) Init(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE notes (
			id   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			text TEXT NOT NULL
		)`)
	if err != nil {
		return fmt.Errorf("creating schema: %w", err)
	}
	return nil
}

// Add inserts a note and returns its generated id (Postgres RETURNING).
func (s *NoteStore) Add(ctx context.Context, text string) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO notes (text) VALUES ($1) RETURNING id`, text).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("inserting note: %w", err)
	}
	return id, nil
}

// List returns all note texts in insertion order.
func (s *NoteStore) List(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT text FROM notes ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("querying notes: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var notes []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return nil, fmt.Errorf("scanning note: %w", err)
		}
		notes = append(notes, text)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating notes: %w", err)
	}
	return notes, nil
}

func main() {
	fmt.Println("this example's demonstration lives in its tests — run `make test` (requires Docker)")
}
