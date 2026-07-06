package main

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// --- test doubles -----------------------------------------------------------

// stubUserStore is a STUB: it returns canned answers and records nothing.
// Stubs are for dependencies whose output the test needs but whose usage it
// doesn't verify.
type stubUserStore struct {
	user User
	err  error
}

func (s stubUserStore) GetUser(context.Context, string) (User, error) {
	return s.user, s.err
}

// mockMailer is a MOCK: it records how it was called so the test can assert
// on the interaction (who was emailed, how many times), and can be programmed
// to fail.
type mockMailer struct {
	mu    sync.Mutex
	calls []sendCall
	err   error
}

type sendCall struct {
	To, Subject, Body string
}

func (m *mockMailer) Send(_ context.Context, to, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, sendCall{To: to, Subject: subject, Body: body})
	return m.err
}

// fakeUserStore is a FAKE: a real, working implementation with a shortcut
// (a map instead of a database). Fakes shine when many tests need consistent,
// stateful behavior rather than one canned answer.
type fakeUserStore struct {
	mu    sync.Mutex
	users map[string]User
}

func newFakeUserStore(users ...User) *fakeUserStore {
	f := &fakeUserStore{users: make(map[string]User)}
	for _, u := range users {
		f.users[u.ID] = u
	}
	return f
}

func (f *fakeUserStore) GetUser(_ context.Context, id string) (User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	user, ok := f.users[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return user, nil
}

// --- tests ------------------------------------------------------------------

func TestWelcomeSendsEmail(t *testing.T) {
	t.Parallel()
	store := stubUserStore{user: User{ID: "42", Name: "Ada", Email: "ada@example.com"}}
	mailer := &mockMailer{}
	svc := NewWelcomeService(store, mailer)

	if err := svc.Welcome(t.Context(), "42"); err != nil {
		t.Fatalf("Welcome: %v", err)
	}

	if len(mailer.calls) != 1 {
		t.Fatalf("mailer called %d times, want 1", len(mailer.calls))
	}
	call := mailer.calls[0]
	if call.To != "ada@example.com" {
		t.Errorf("sent to %q, want %q", call.To, "ada@example.com")
	}
	if call.Subject != "Welcome!" {
		t.Errorf("subject %q, want %q", call.Subject, "Welcome!")
	}
}

func TestWelcomeStoreFailureSkipsEmail(t *testing.T) {
	t.Parallel()
	store := stubUserStore{err: ErrUserNotFound}
	mailer := &mockMailer{}
	svc := NewWelcomeService(store, mailer)

	err := svc.Welcome(t.Context(), "missing")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Welcome error = %v, want wrapped ErrUserNotFound", err)
	}
	if len(mailer.calls) != 0 {
		t.Errorf("mailer called %d times, want 0 — no email on lookup failure", len(mailer.calls))
	}
}

func TestWelcomeWrapsMailerError(t *testing.T) {
	t.Parallel()
	errSMTP := errors.New("smtp: connection refused")
	store := stubUserStore{user: User{ID: "42", Name: "Ada", Email: "ada@example.com"}}
	mailer := &mockMailer{err: errSMTP}
	svc := NewWelcomeService(store, mailer)

	err := svc.Welcome(t.Context(), "42")
	if !errors.Is(err, errSMTP) {
		t.Fatalf("Welcome error = %v, want wrapped %v", err, errSMTP)
	}
}

// TestWelcomeWithFuncDouble shows the function-adapter trick: for a one-off
// behavior, a MailerFunc closure is a complete test double — no type needed.
func TestWelcomeWithFuncDouble(t *testing.T) {
	t.Parallel()
	store := stubUserStore{user: User{ID: "7", Name: "Lin", Email: "lin@example.com"}}

	var gotBody string
	mailer := MailerFunc(func(_ context.Context, _, _, body string) error {
		gotBody = body
		return nil
	})

	if err := NewWelcomeService(store, mailer).Welcome(t.Context(), "7"); err != nil {
		t.Fatalf("Welcome: %v", err)
	}
	if want := "Hi Lin, thanks for signing up."; gotBody != want {
		t.Errorf("body = %q, want %q", gotBody, want)
	}
}

// TestWelcomeWithFake exercises the service against the stateful fake:
// present and missing users behave like a real store would.
func TestWelcomeWithFake(t *testing.T) {
	t.Parallel()
	store := newFakeUserStore(User{ID: "1", Name: "Grace", Email: "grace@example.com"})
	mailer := &mockMailer{}
	svc := NewWelcomeService(store, mailer)

	if err := svc.Welcome(t.Context(), "1"); err != nil {
		t.Fatalf("Welcome existing user: %v", err)
	}
	if err := svc.Welcome(t.Context(), "2"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Welcome missing user error = %v, want ErrUserNotFound", err)
	}
	if len(mailer.calls) != 1 {
		t.Errorf("mailer called %d times, want exactly 1", len(mailer.calls))
	}
}
