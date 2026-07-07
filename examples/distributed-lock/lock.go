package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrNotAcquired is returned when another holder owns the lock.
var ErrNotAcquired = errors.New("lock is held by someone else")

// ErrNotHeld is returned by Release/Extend when the caller no longer owns the
// lock (it expired, or was never theirs) — the operation did nothing.
var ErrNotHeld = errors.New("lock not held by this token")

// releaseScript deletes the lock only if the caller still owns it. The
// GET-compare and DEL must be atomic: a plain GET+DEL from the client races
// with expiry and can delete the *next* holder's lock.
var releaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0`)

// extendScript renews the TTL only if the caller still owns the lock — the
// heartbeat half of a lease.
var extendScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0`)

// Lock is a single-Redis distributed lock with an owner token and a fencing
// token. The owner token proves identity to Redis (safe release/extend); the
// fencing token proves *freshness* to downstream systems (stale holders get
// rejected even after their lock silently expired).
type Lock struct {
	client *redis.Client
	key    string
	token  string
	// Fence is a monotonically increasing acquisition number. Downstream
	// resources must reject writes carrying a fence lower than the highest
	// they have seen.
	Fence int64
}

// Acquire tries to take the lock for ttl. It returns ErrNotAcquired without
// waiting if another holder owns it.
func Acquire(ctx context.Context, client *redis.Client, key string, ttl time.Duration) (*Lock, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("generating owner token: %w", err)
	}
	token := hex.EncodeToString(buf)

	// SET NX PX is the atomic "create if absent, with expiry" primitive.
	// The TTL guarantees liveness: a crashed holder can't wedge the system.
	ok, err := client.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return nil, fmt.Errorf("acquiring lock: %w", err)
	}
	if !ok {
		return nil, ErrNotAcquired
	}

	// The fence counter only moves forward, one increment per acquisition,
	// so a later holder always carries a larger number.
	fence, err := client.Incr(ctx, key+":fence").Result()
	if err != nil {
		return nil, fmt.Errorf("issuing fencing token: %w", err)
	}
	return &Lock{client: client, key: key, token: token, Fence: fence}, nil
}

// Release frees the lock if this holder still owns it.
func (l *Lock) Release(ctx context.Context) error {
	deleted, err := releaseScript.Run(ctx, l.client, []string{l.key}, l.token).Int()
	if err != nil {
		return fmt.Errorf("releasing lock: %w", err)
	}
	if deleted == 0 {
		return ErrNotHeld
	}
	return nil
}

// Extend renews the lease for another ttl if this holder still owns the lock.
// Long-running holders call this periodically (well before the TTL) so the
// lock survives exactly as long as the work does.
func (l *Lock) Extend(ctx context.Context, ttl time.Duration) error {
	extended, err := extendScript.Run(ctx, l.client, []string{l.key}, l.token, ttl.Milliseconds()).Int()
	if err != nil {
		return fmt.Errorf("extending lock: %w", err)
	}
	if extended == 0 {
		return ErrNotHeld
	}
	return nil
}
