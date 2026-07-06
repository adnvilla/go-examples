package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- handler tests: httptest.NewRecorder, no network involved ---

// TestHealthz shows the minimal recorder pattern: build an in-memory request,
// serve it into a ResponseRecorder, and assert on the recorded response.
func TestHealthz(t *testing.T) {
	t.Parallel()
	mux := NewUserServer().Mux()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "ok\n" {
		t.Errorf("GET /healthz body = %q, want %q", got, "ok\n")
	}
}

// TestCreateAndGetUser drives two handlers through the same mux, showing that
// recorder-based tests exercise real routing (method match, path parameters)
// — not just the handler function in isolation.
func TestCreateAndGetUser(t *testing.T) {
	t.Parallel()
	mux := NewUserServer().Mux()

	// POST /users
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/users", strings.NewReader(`{"name":"Grace"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /users status = %d, want %d (body: %s)", rec.Code, http.StatusCreated, rec.Body)
	}

	// GET /users/1 — the id assigned to the first created user
	req = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/users/1", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /users/1 status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got, want := rec.Body.String(), `{"id":"1","name":"Grace"}`+"\n"; got != want {
		t.Errorf("GET /users/1 body = %q, want %q", got, want)
	}
}

func TestCreateUserRejectsBadBody(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
	}{
		{"not json", "not-json"},
		{"empty name", `{"name":""}`},
		{"empty body", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mux := NewUserServer().Mux()

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/users", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("POST /users with %s: status = %d, want %d", tc.name, rec.Code, http.StatusBadRequest)
			}
		})
	}
}

// --- client tests: httptest.NewServer, a real HTTP server on a loopback port ---

// TestFetchUser points the real client function at a real (loopback) server:
// the full stack runs — TCP, http.Client, routing, JSON — with no fixed port
// and automatic cleanup.
func TestFetchUser(t *testing.T) {
	t.Parallel()
	server := NewUserServer()
	ts := httptest.NewServer(server.Mux())
	defer ts.Close()

	// Seed a user through the API itself.
	seed, err := http.NewRequestWithContext(t.Context(), http.MethodPost, ts.URL+"/users", strings.NewReader(`{"name":"Ada"}`))
	if err != nil {
		t.Fatalf("building seed request: %v", err)
	}
	seed.Header.Set("Content-Type", "application/json")
	resp, err := ts.Client().Do(seed)
	if err != nil {
		t.Fatalf("seeding user: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	user, err := FetchUser(t.Context(), ts.Client(), ts.URL, "1")
	if err != nil {
		t.Fatalf("FetchUser: %v", err)
	}
	if user.Name != "Ada" {
		t.Errorf("FetchUser name = %q, want %q", user.Name, "Ada")
	}
}

// TestFetchUserErrors uses a throwaway handler to force server-side failures
// the client must handle — no need for the real UserServer at all.
func TestFetchUserErrors(t *testing.T) {
	t.Parallel()

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		ts := httptest.NewServer(NewUserServer().Mux())
		defer ts.Close()

		if _, err := FetchUser(t.Context(), ts.Client(), ts.URL, "999"); err == nil {
			t.Fatal("FetchUser for missing user: want error, got nil")
		}
	})

	t.Run("server error", func(t *testing.T) {
		t.Parallel()
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		}))
		defer ts.Close()

		if _, err := FetchUser(t.Context(), ts.Client(), ts.URL, "1"); err == nil {
			t.Fatal("FetchUser against failing server: want error, got nil")
		}
	})

	t.Run("garbage body", func(t *testing.T) {
		t.Parallel()
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, "not json")
		}))
		defer ts.Close()

		if _, err := FetchUser(t.Context(), ts.Client(), ts.URL, "1"); err == nil {
			t.Fatal("FetchUser with non-JSON body: want error, got nil")
		}
	})
}
