// HTTP client patterns: timeouts, retries with backoff, context cancellation,
// and JSON decoding. Uses only the standard library.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"time"
)

// Client wraps http.Client with retry logic.
type Client struct {
	http       *http.Client
	maxRetries int
	baseDelay  time.Duration
}

func NewClient(timeout time.Duration, maxRetries int) *Client {
	return &Client{
		http:       &http.Client{Timeout: timeout},
		maxRetries: maxRetries,
		baseDelay:  200 * time.Millisecond,
	}
}

// GetJSON fetches a URL and decodes the JSON body into dst.
// It retries on 5xx responses and network errors, backing off exponentially.
func (c *Client) GetJSON(ctx context.Context, url string, dst any) error {
	var lastErr error
	for attempt := range c.maxRetries + 1 {
		if attempt > 0 {
			delay := time.Duration(float64(c.baseDelay) * math.Pow(2, float64(attempt-1)))
			slog.Info("retrying", "attempt", attempt, "delay", delay, "url", url)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during backoff: %w", ctx.Err())
			}
		}

		if err := c.doGet(ctx, url, dst); err != nil {
			lastErr = err
			var retryable *retryableError
			if !errors.As(err, &retryable) {
				return err // non-retryable: propagate immediately
			}
			slog.Warn("retryable error", "error", err)
			continue
		}
		return nil
	}
	return fmt.Errorf("all %d attempts failed: %w", c.maxRetries+1, lastErr)
}

type retryableError struct{ cause error }

func (e *retryableError) Error() string { return fmt.Sprintf("retryable: %v", e.cause) }
func (e *retryableError) Unwrap() error { return e.cause }

func (c *Client) doGet(ctx context.Context, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return &retryableError{cause: err}
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= 500 {
		return &retryableError{cause: fmt.Errorf("server error: %s", resp.Status)}
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("client error: %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

type Post struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

func main() {
	client := NewClient(5*time.Second, 3)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var post Post
	url := "https://jsonplaceholder.typicode.com/posts/1"
	if err := client.GetJSON(ctx, url, &post); err != nil {
		slog.Error("request failed", "error", err)
		return
	}

	fmt.Printf("Post #%d: %s\n", post.ID, post.Title)
}
