// Demonstrates a distributed lock over a single Redis: atomic acquisition
// with SET NX PX, owner tokens so only the holder can release or extend
// (via Lua compare-and-act scripts), lease renewal for long work, and — the
// part naive implementations skip — fencing tokens, which protect the data
// even after a lock silently expires under a paused holder.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

const lockKey = "examples:distributed-lock:orders"

// storage simulates the downstream resource the lock protects. It keeps the
// highest fencing token it has accepted and rejects anything older — the
// last line of defense when a stale lock holder comes back from the dead.
type storage struct {
	highestFence int64
}

func (s *storage) write(fence int64, what string) error {
	if fence < s.highestFence {
		return fmt.Errorf("rejected: fence %d is stale (highest seen: %d)", fence, s.highestFence)
	}
	s.highestFence = fence
	fmt.Printf("  storage: accepted %q with fence %d\n", what, fence)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer client.Close() //nolint:errcheck
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis is not reachable on localhost:6379 (docker compose up -d redis): %w", err)
	}
	// Start from a clean slate so repeated runs behave identically.
	if err := client.Del(ctx, lockKey, lockKey+":fence").Err(); err != nil {
		return fmt.Errorf("cleaning previous state: %w", err)
	}

	store := &storage{}

	if err := mutualExclusion(ctx, client); err != nil {
		return err
	}
	if err := leaseRenewal(ctx, client, store); err != nil {
		return err
	}
	return staleHolder(ctx, client, store)
}

// mutualExclusion shows the basic guarantee: while A holds the lock, B's
// acquisition fails fast instead of proceeding into the critical section.
func mutualExclusion(ctx context.Context, client *redis.Client) error {
	fmt.Println("--- mutual exclusion: one holder at a time ---")

	lockA, err := Acquire(ctx, client, lockKey, time.Second)
	if err != nil {
		return fmt.Errorf("worker A acquiring: %w", err)
	}
	fmt.Printf("worker A: acquired (fence %d)\n", lockA.Fence)

	if _, err := Acquire(ctx, client, lockKey, time.Second); !errors.Is(err, ErrNotAcquired) {
		return fmt.Errorf("worker B: expected ErrNotAcquired, got %w", err)
	}
	fmt.Println("worker B: denied — lock is held")

	if err := lockA.Release(ctx); err != nil {
		return fmt.Errorf("worker A releasing: %w", err)
	}
	fmt.Println("worker A: released")

	lockB, err := Acquire(ctx, client, lockKey, time.Second)
	if err != nil {
		return fmt.Errorf("worker B acquiring after release: %w", err)
	}
	fmt.Printf("worker B: acquired after release (fence %d)\n", lockB.Fence)
	return lockB.Release(ctx)
}

// leaseRenewal shows the heartbeat: work that outlives the TTL keeps the
// lock by extending it, and the lock still ends the moment the work does.
func leaseRenewal(ctx context.Context, client *redis.Client, store *storage) error {
	fmt.Println("\n--- lease renewal: work longer than the TTL ---")

	lock, err := Acquire(ctx, client, lockKey, 200*time.Millisecond)
	if err != nil {
		return fmt.Errorf("acquiring: %w", err)
	}
	fmt.Printf("worker: acquired with 200ms TTL (fence %d)\n", lock.Fence)

	// Three 150ms work slices — without renewal the 200ms lease would lapse
	// mid-way; extending well before expiry keeps ownership continuous.
	for slice := 1; slice <= 3; slice++ {
		time.Sleep(150 * time.Millisecond)
		if err := lock.Extend(ctx, 200*time.Millisecond); err != nil {
			return fmt.Errorf("extending after slice %d: %w", slice, err)
		}
		fmt.Printf("worker: slice %d done, lease extended\n", slice)
	}

	if err := store.write(lock.Fence, "renewal result"); err != nil {
		return err
	}
	if err := lock.Release(ctx); err != nil {
		return fmt.Errorf("releasing: %w", err)
	}
	fmt.Println("worker: released after ~450ms of work on a 200ms lease")
	return nil
}

// staleHolder is the scenario fencing tokens exist for: C's lock expires
// while C is paused (a long GC pause, a network partition), D legitimately
// acquires, and then C wakes up still believing it holds the lock. Redis
// refuses C's release (wrong token) and — crucially — storage refuses C's
// write (stale fence).
func staleHolder(ctx context.Context, client *redis.Client, store *storage) error {
	fmt.Println("\n--- stale holder: expiry + fencing tokens ---")

	lockC, err := Acquire(ctx, client, lockKey, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("worker C acquiring: %w", err)
	}
	fmt.Printf("worker C: acquired with 100ms TTL (fence %d)\n", lockC.Fence)

	fmt.Println("worker C: pauses for 250ms (GC pause / network partition)...")
	time.Sleep(250 * time.Millisecond) // the lock expires during this pause

	lockD, err := Acquire(ctx, client, lockKey, time.Second)
	if err != nil {
		return fmt.Errorf("worker D acquiring after expiry: %w", err)
	}
	fmt.Printf("worker D: acquired the expired lock (fence %d)\n", lockD.Fence)
	if err := store.write(lockD.Fence, "D's update"); err != nil {
		return err
	}

	// C wakes up. It cannot release D's lock...
	if err := lockC.Release(ctx); !errors.Is(err, ErrNotHeld) {
		return fmt.Errorf("worker C release: expected ErrNotHeld, got %w", err)
	}
	fmt.Println("worker C: wakes; release refused (owner token no longer matches)")

	// ...and its write is rejected by the fence check, protecting the data.
	writeErr := store.write(lockC.Fence, "C's late update")
	if writeErr == nil {
		return errors.New("storage accepted a stale fence — fencing is broken")
	}
	fmt.Printf("worker C: write %v\n", writeErr)

	if err := lockD.Release(ctx); err != nil {
		return fmt.Errorf("worker D releasing: %w", err)
	}
	fmt.Println("worker D: released cleanly")
	return nil
}
