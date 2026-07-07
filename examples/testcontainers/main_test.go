package main

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver "pgx"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// startPostgres boots a fresh PostgreSQL container for this one test and
// returns an initialized store. Everything is cleaned up by t.Cleanup — the
// test owns its infrastructure from first byte to last.
func startPostgres(t *testing.T) *NoteStore {
	t.Helper()
	// Skip (rather than fail) on machines without a working Docker daemon.
	testcontainers.SkipIfProviderIsNotHealthy(t)

	container, err := tcpostgres.Run(t.Context(), "postgres:16-alpine",
		tcpostgres.WithDatabase("examples"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("secret"),
		testcontainers.WithWaitStrategy(
			// "ready to accept connections" appears once during initdb and
			// once for the real server — waiting for the second occurrence
			// is the documented readiness signal for this image.
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("starting postgres container: %v", err)
	}
	testcontainers.CleanupContainer(t, container)

	dsn, err := container.ConnectionString(t.Context(), "sslmode=disable")
	if err != nil {
		t.Fatalf("building connection string: %v", err)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	store := NewNoteStore(db)
	if err := store.Init(t.Context()); err != nil {
		t.Fatalf("initializing schema: %v", err)
	}
	return store
}

func TestAddAndList(t *testing.T) {
	t.Parallel()
	store := startPostgres(t)

	first, err := store.Add(t.Context(), "buy milk")
	if err != nil {
		t.Fatalf("adding first note: %v", err)
	}
	second, err := store.Add(t.Context(), "write tests")
	if err != nil {
		t.Fatalf("adding second note: %v", err)
	}
	if second <= first {
		t.Errorf("generated ids not increasing: %d then %d", first, second)
	}

	notes, err := store.List(t.Context())
	if err != nil {
		t.Fatalf("listing notes: %v", err)
	}
	want := []string{"buy milk", "write tests"}
	if len(notes) != len(want) {
		t.Fatalf("notes = %v, want %v", notes, want)
	}
	for i := range want {
		if notes[i] != want[i] {
			t.Errorf("notes[%d] = %q, want %q", i, notes[i], want[i])
		}
	}
}

// TestEachTestGetsAFreshDatabase proves the isolation property that makes
// container-per-test worth its startup cost: nothing from any other test —
// including TestAddAndList running in parallel — is visible here.
func TestEachTestGetsAFreshDatabase(t *testing.T) {
	t.Parallel()
	store := startPostgres(t)

	notes, err := store.List(t.Context())
	if err != nil {
		t.Fatalf("listing notes: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("fresh container has %d notes (%v), want 0 — isolation broken", len(notes), notes)
	}
}
