// Demonstrates database/sql fundamentals against an embedded SQLite database
// (modernc.org/sqlite — pure Go, no CGo, no server): schema setup, inserts
// with placeholders, multi-row queries with the rows.Next/Scan/Err discipline,
// single-row lookups with sql.ErrNoRows, and transactions with rollback.
// Everything runs in memory, so the example needs no Docker and cleans up
// after itself.
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

type Book struct {
	ID    int64
	Title string
	Year  int
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// sql.Open validates arguments but doesn't connect; the pool dials lazily.
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close() //nolint:errcheck

	// With ":memory:", every pool connection would get its OWN empty database.
	// Capping the pool at one connection makes the in-memory DB behave like a
	// single shared database. (File-backed DSNs don't need this.)
	db.SetMaxOpenConns(1)

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("pinging database: %w", err)
	}

	if err := createSchema(ctx, db); err != nil {
		return err
	}
	if err := insertBooks(ctx, db); err != nil {
		return err
	}
	if err := queryBooks(ctx, db); err != nil {
		return err
	}
	if err := querySingleBook(ctx, db); err != nil {
		return err
	}
	return transactions(ctx, db)
}

func createSchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE books (
			id    INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			year  INTEGER NOT NULL
		)`)
	if err != nil {
		return fmt.Errorf("creating schema: %w", err)
	}
	fmt.Println("--- schema created ---")
	return nil
}

// insertBooks shows parameterized Exec: placeholders keep values out of the
// SQL text (no injection, no quoting bugs), and the result reports the
// generated key and affected-row count.
func insertBooks(ctx context.Context, db *sql.DB) error {
	fmt.Println("\n--- inserts with placeholders ---")
	books := []Book{
		{Title: "The Go Programming Language", Year: 2015},
		{Title: "Go in Action", Year: 2015},
		{Title: "Learning Go", Year: 2021},
	}
	for _, b := range books {
		res, err := db.ExecContext(ctx,
			`INSERT INTO books (title, year) VALUES (?, ?)`, b.Title, b.Year)
		if err != nil {
			return fmt.Errorf("inserting %q: %w", b.Title, err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("reading generated id: %w", err)
		}
		fmt.Printf("inserted id=%d %q (%d)\n", id, b.Title, b.Year)
	}
	return nil
}

// queryBooks shows the multi-row discipline: Next, Scan, Close — and the
// often-forgotten rows.Err(), which is where a mid-iteration failure surfaces.
func queryBooks(ctx context.Context, db *sql.DB) error {
	fmt.Println("\n--- multi-row query ---")
	rows, err := db.QueryContext(ctx,
		`SELECT id, title, year FROM books WHERE year = ? ORDER BY title`, 2015)
	if err != nil {
		return fmt.Errorf("querying books: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Year); err != nil {
			return fmt.Errorf("scanning row: %w", err)
		}
		fmt.Printf("published in 2015: %q\n", b.Title)
	}
	// rows.Next returning false means "stopped": rows.Err distinguishes
	// end-of-results from a query that died partway through.
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating rows: %w", err)
	}
	return nil
}

// querySingleBook shows QueryRow: Scan returns sql.ErrNoRows for an empty
// result, which is a normal condition to branch on, not an outage.
func querySingleBook(ctx context.Context, db *sql.DB) error {
	fmt.Println("\n--- single-row query and ErrNoRows ---")
	var b Book
	err := db.QueryRowContext(ctx,
		`SELECT id, title, year FROM books WHERE year > ?`, 2019).
		Scan(&b.ID, &b.Title, &b.Year)
	if err != nil {
		return fmt.Errorf("querying newest book: %w", err)
	}
	fmt.Printf("published after 2019: %q\n", b.Title)

	err = db.QueryRowContext(ctx,
		`SELECT id, title, year FROM books WHERE year > ?`, 2100).
		Scan(&b.ID, &b.Title, &b.Year)
	if errors.Is(err, sql.ErrNoRows) {
		fmt.Println("published after 2100: none (sql.ErrNoRows)")
	} else if err != nil {
		return fmt.Errorf("querying future book: %w", err)
	}
	return nil
}

// transactions shows BeginTx with both outcomes: a committed insert that
// persists and a rolled-back insert that leaves no trace.
func transactions(ctx context.Context, db *sql.DB) error {
	fmt.Println("\n--- transactions: commit and rollback ---")

	// Committed transaction.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning tx: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO books (title, year) VALUES (?, ?)`, "100 Go Mistakes", 2022); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("inserting in tx: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing: %w", err)
	}

	// Rolled-back transaction: the insert happens inside the tx, then vanishes.
	tx, err = db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning tx: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO books (title, year) VALUES (?, ?)`, "Draft Never Published", 2026); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("inserting in tx: %w", err)
	}
	if err := tx.Rollback(); err != nil {
		return fmt.Errorf("rolling back: %w", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM books`).Scan(&count); err != nil {
		return fmt.Errorf("counting books: %w", err)
	}
	fmt.Printf("after commit + rollback: %d books (the rolled-back insert left no trace)\n", count)
	return nil
}
