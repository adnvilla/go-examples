// Structured logging with log/slog (Go 1.21).
// slog replaces ad-hoc log.Printf calls with structured key-value pairs
// that JSON handlers can forward to log aggregators without post-processing.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"
)

// requestID is a typed context key to avoid collisions.
type requestID string

const reqIDKey requestID = "reqID"

// withRequestID adds a request ID to the logger stored in ctx.
func withRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, reqIDKey, id)
}

// logger extracts request-scoped fields from ctx and returns a child logger.
func logger(ctx context.Context) *slog.Logger {
	l := slog.Default()
	if id, ok := ctx.Value(reqIDKey).(string); ok {
		l = l.With("request_id", id)
	}
	return l
}

func processOrder(ctx context.Context, orderID int) error {
	log := logger(ctx)
	log.Info("processing order", "order_id", orderID)

	start := time.Now()
	time.Sleep(10 * time.Millisecond) // simulate work
	log.Info("order processed", "order_id", orderID, "duration_ms", time.Since(start).Milliseconds())
	return nil
}

func main() {
	// Text handler — human-readable, default to stderr.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	slog.Info("application started", "version", "1.0.0")
	slog.Debug("debug message — only visible at Debug level")

	// Structured fields — the key-value pairs appear consistently in every log line.
	slog.Warn("high memory usage", "used_mb", 512, "limit_mb", 1024, "pct", 50)

	// Group related fields to avoid key collisions.
	slog.Info("database connected",
		slog.Group("db", "host", "localhost", "port", 5432, "name", "orders"),
	)

	// Context-aware logging: request ID propagates automatically.
	ctx := withRequestID(context.Background(), "req-abc-123")
	if err := processOrder(ctx, 42); err != nil {
		logger(ctx).Error("order failed", "error", err)
	}

	// Switch to JSON handler — same API, structured output for log aggregators.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
	slog.Info("switched to JSON handler", "env", "production")
}
