// Demonstrates testing HTTP handlers and clients with net/http/httptest:
// httptest.NewRecorder unit-tests a handler with no network involved, and
// httptest.NewServer gives a client a real loopback server to talk to — no
// mocks, no port conflicts, no external processes.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
)

// User is the resource served by the small API under test.
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// UserServer is an in-memory user store exposed over HTTP — just enough
// surface (a POST, a GET with a path parameter, JSON in and out, a 404 path)
// to make the tests representative of a real handler.
type UserServer struct {
	mu     sync.Mutex
	nextID int
	users  map[string]User
}

// NewUserServer returns an empty store.
func NewUserServer() *UserServer {
	return &UserServer{nextID: 1, users: make(map[string]User)}
}

// Mux wires the routes using Go 1.22 method-and-pattern routing.
func (s *UserServer) Mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("POST /users", s.handleCreateUser)
	mux.HandleFunc("GET /users/{id}", s.handleGetUser)
	return mux
}

func (s *UserServer) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprintln(w, "ok"); err != nil {
		return // response already committed; a real service would log this
	}
}

func (s *UserServer) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Name == "" {
		http.Error(w, "body must be JSON with a non-empty name", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	user := User{ID: strconv.Itoa(s.nextID), Name: in.Name}
	s.nextID++
	s.users[user.ID] = user
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *UserServer) handleGetUser(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	user, ok := s.users[r.PathValue("id")]
	s.mu.Unlock()
	if !ok {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// FetchUser is the client side under test: it GETs /users/{id} from baseURL
// and decodes the response. Tests point baseURL at an httptest.Server.
func FetchUser(ctx context.Context, client *http.Client, baseURL, id string) (User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/users/"+id, nil)
	if err != nil {
		return User{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return User{}, err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return User{}, fmt.Errorf("GET %s: unexpected status %s", req.URL, resp.Status)
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return User{}, fmt.Errorf("decoding response: %w", err)
	}
	return user, nil
}

func main() {
	fmt.Println("this example's demonstration lives in its tests — run `make test` (or `go test -v ./...`)")
}
