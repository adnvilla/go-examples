package main

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// requireLocalRedis skips unless REDIS_LOCAL=1 (same convention as the
// dynamodb example) and returns a client with a test-scoped key.
func requireLocalRedis(t *testing.T) (*redis.Client, string) {
	t.Helper()
	if os.Getenv("REDIS_LOCAL") == "" {
		t.Skip("set REDIS_LOCAL=1 to run Redis integration tests (requires local Redis on :6379)")
	}

	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := client.Ping(t.Context()).Err(); err != nil {
		t.Fatalf("redis not reachable: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	key := "examples:distributed-lock:test:" + t.Name()
	if err := client.Del(t.Context(), key, key+":fence").Err(); err != nil {
		t.Fatalf("cleaning test keys: %v", err)
	}
	return client, key
}

func TestMutualExclusion(t *testing.T) {
	t.Parallel()
	client, key := requireLocalRedis(t)

	lock, err := Acquire(t.Context(), client, key, time.Second)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	if _, err := Acquire(t.Context(), client, key, time.Second); !errors.Is(err, ErrNotAcquired) {
		t.Fatalf("second acquire: err = %v, want ErrNotAcquired", err)
	}
	if err := lock.Release(t.Context()); err != nil {
		t.Fatalf("release: %v", err)
	}
	if _, err := Acquire(t.Context(), client, key, time.Second); err != nil {
		t.Fatalf("acquire after release: %v", err)
	}
}

func TestReleaseIsOwnerOnly(t *testing.T) {
	t.Parallel()
	client, key := requireLocalRedis(t)

	stale, err := Acquire(t.Context(), client, key, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	time.Sleep(100 * time.Millisecond) // let it expire

	current, err := Acquire(t.Context(), client, key, time.Second)
	if err != nil {
		t.Fatalf("acquire after expiry: %v", err)
	}

	// The stale holder must not be able to delete the new holder's lock.
	if err := stale.Release(t.Context()); !errors.Is(err, ErrNotHeld) {
		t.Fatalf("stale release: err = %v, want ErrNotHeld", err)
	}
	if err := current.Release(t.Context()); err != nil {
		t.Fatalf("current holder release: %v", err)
	}
}

func TestExtendKeepsLockAlive(t *testing.T) {
	t.Parallel()
	client, key := requireLocalRedis(t)

	lock, err := Acquire(t.Context(), client, key, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	for range 3 {
		time.Sleep(60 * time.Millisecond)
		if err := lock.Extend(t.Context(), 100*time.Millisecond); err != nil {
			t.Fatalf("extend: %v", err)
		}
	}
	// 180ms elapsed on a 100ms TTL — only renewal explains still holding it.
	if _, err := Acquire(t.Context(), client, key, time.Second); !errors.Is(err, ErrNotAcquired) {
		t.Fatalf("acquire during extended lease: err = %v, want ErrNotAcquired", err)
	}
	if err := lock.Release(t.Context()); err != nil {
		t.Fatalf("release: %v", err)
	}
}

func TestFencingTokensAreMonotonic(t *testing.T) {
	t.Parallel()
	client, key := requireLocalRedis(t)

	var last int64
	for i := range 3 {
		lock, err := Acquire(t.Context(), client, key, time.Second)
		if err != nil {
			t.Fatalf("acquire %d: %v", i, err)
		}
		if lock.Fence <= last {
			t.Fatalf("fence %d not greater than previous %d", lock.Fence, last)
		}
		last = lock.Fence
		if err := lock.Release(t.Context()); err != nil {
			t.Fatalf("release %d: %v", i, err)
		}
	}
}
