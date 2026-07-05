// Error handling patterns: sentinel errors, custom error types, wrapping and unwrapping.
// errors.Is checks identity; errors.As unwraps to a concrete type.
// %w in fmt.Errorf wraps an error so both functions can traverse the chain.
package main

import (
	"errors"
	"fmt"
)

// --- sentinel errors ---

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
)

// --- custom error type ---

// ValidationError carries the field name and a description of the constraint violation.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed on %q: %s", e.Field, e.Message)
}

// --- a layered call chain ---

func findUser(id int) error {
	if id <= 0 {
		return &ValidationError{Field: "id", Message: "must be positive"}
	}
	if id > 100 {
		// Wrap the sentinel so callers can inspect it with errors.Is.
		return fmt.Errorf("findUser(%d): %w", id, ErrNotFound)
	}
	return nil
}

func loadProfile(userID int) error {
	if err := findUser(userID); err != nil {
		// Wrap again: the original error is still reachable via Unwrap.
		return fmt.Errorf("loadProfile: %w", err)
	}
	return nil
}

func main() {
	// Case 1: sentinel error — errors.Is traverses the wrap chain.
	err := loadProfile(999)
	fmt.Println("is ErrNotFound:", errors.Is(err, ErrNotFound))
	fmt.Println("error:", err)
	fmt.Println()

	// Case 2: custom error type — errors.As unwraps to *ValidationError.
	err = loadProfile(-1)
	var valErr *ValidationError
	if errors.As(err, &valErr) {
		fmt.Printf("validation error on field %q: %s\n", valErr.Field, valErr.Message)
	}
	fmt.Println()

	// Case 3: no error.
	err = loadProfile(42)
	fmt.Println("no error:", err)
	fmt.Println()

	// Case 4: joining multiple errors (Go 1.20+).
	errs := []error{ErrNotFound, ErrForbidden}
	joined := errors.Join(errs...)
	fmt.Println("joined:", joined)
	fmt.Println("is ErrNotFound:", errors.Is(joined, ErrNotFound))
	fmt.Println("is ErrForbidden:", errors.Is(joined, ErrForbidden))
}
