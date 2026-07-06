// Demonstrates hand-rolled test doubles (stubs, mocks, fakes) built on plain
// Go interfaces — the reason to accept interfaces at dependency seams is that
// tests can then substitute behavior with a few lines of code and no framework.
package main

import (
	"context"
	"errors"
	"fmt"
)

// User is the domain entity the service works with.
type User struct {
	ID    string
	Name  string
	Email string
}

// ErrUserNotFound is returned by UserStore implementations when the id is unknown.
var ErrUserNotFound = errors.New("user not found")

// UserStore is the persistence seam. The service defines the interface it
// *consumes* (Go style: interfaces belong to the consumer, not the implementer),
// so any storage — Postgres, DynamoDB, or a test double — can satisfy it.
type UserStore interface {
	GetUser(ctx context.Context, id string) (User, error)
}

// Mailer is the outbound-delivery seam.
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// MailerFunc adapts a plain function to the Mailer interface — the http.HandlerFunc
// trick. Tests can define one-off mailers inline without declaring a type.
type MailerFunc func(ctx context.Context, to, subject, body string) error

// Send calls f.
func (f MailerFunc) Send(ctx context.Context, to, subject, body string) error {
	return f(ctx, to, subject, body)
}

// WelcomeService is the unit under test: look a user up, send them a welcome
// email, and translate failures from either dependency into wrapped errors.
type WelcomeService struct {
	store  UserStore
	mailer Mailer
}

// NewWelcomeService wires the service's dependencies.
func NewWelcomeService(store UserStore, mailer Mailer) *WelcomeService {
	return &WelcomeService{store: store, mailer: mailer}
}

// Welcome sends the welcome email for the given user id.
func (s *WelcomeService) Welcome(ctx context.Context, userID string) error {
	user, err := s.store.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("looking up user %s: %w", userID, err)
	}

	subject := "Welcome!"
	body := fmt.Sprintf("Hi %s, thanks for signing up.", user.Name)
	if err := s.mailer.Send(ctx, user.Email, subject, body); err != nil {
		return fmt.Errorf("sending welcome email to %s: %w", user.Email, err)
	}
	return nil
}

func main() {
	fmt.Println("this example's demonstration lives in its tests — run `make test` (or `go test -v ./...`)")
}
