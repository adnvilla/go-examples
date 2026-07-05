# Redis Task Queue

**Category:** database
**Difficulty:** Advanced

## Objective

Show a Redis-backed task queue: an HTTP endpoint enqueues jobs and blocks waiting for results, while background "engine" workers pull jobs off the queue, do some (simulated) work, and publish results back — all coordinated through Redis lists, with go-redis v9's context-first API.

## Concepts Covered

- `redis.NewClient` + `Ping` to verify connectivity at startup, failing fast if Redis is unreachable
- Redis lists as queues: `RPush` to enqueue, `BLPop` (blocking left-pop) to dequeue with a timeout
- A producer/consumer split: `handleQuote` enqueues work and blocks on the *result* key; `runEngine` blocks on the *task* key, works, then pushes to the result key
- Fan-out within a single request: `handleQuote` enqueues to *every* engine's queue concurrently and waits for all their results
- go-redis v9's requirement that every operation take a `context.Context` as its first argument

## Prerequisites

- Go 1.24+
- A running Redis instance (this repo's `docker-compose.yml` provides one on `localhost:6379`):
  ```bash
  docker compose up -d redis
  # or, from this directory:
  make infra-up
  ```

## Project Structure

```
redis/
├── go.mod
├── main.go
├── loadtestk6.js   (k6 load-test script)
└── README.md
```

## How to Run

```bash
make infra-up   # start Redis
make run        # starts the server on :8080
```

In another terminal:
```bash
curl http://localhost:8080/quote
```

Optional load test with [k6](https://k6.io/) (adjust `getUrl()` in `loadtestk6.js` if not running locally):
```bash
k6 run loadtestk6.js
```

## Expected Output

Each request fans out to both engines and waits up to 4 seconds for each; actual delay values are random (0-2s per engine), so exact numbers vary between requests:

```
$ curl http://localhost:8080/quote
{"engine":["engine 0 result: 277","engine 1 result: 1110"]}
```

## Code Walkthrough

- `main` connects to Redis and `Ping`s it immediately — if Redis isn't reachable, the program exits with a clear error rather than starting a server that would fail on every request.
- `runEngine(engineID)` runs forever in its own goroutine (one per engine, `numEngines = 2`): it `BLPop`s its own task queue (`queue:task:<id>`), and on receiving a job ID, sleeps a random delay (simulating work), then `RPush`es the result to a job-specific key (`queue:processed:<id>:<jobID>`).
- `handleQuote` is the HTTP handler: for each engine, it calls `enqueueAndWait` concurrently (one goroutine per engine) and collects all results via a channel, so the total latency is bounded by the *slowest* engine, not the sum of all of them.
- `enqueueAndWait` does the producer side: `Incr` to get a unique job ID for that engine's counter, `RPush` the job ID onto the task queue, then `BLPop` on the *specific* result key for that job ID — this is what lets multiple concurrent requests share the same task queue without their results getting mixed up, since each job gets its own uniquely-keyed result slot.
- Every Redis call takes a `context.Context` as its first argument (`ctx` from the incoming HTTP request in `handleQuote`'s path, `context.Background()` in the long-lived `runEngine` loop) — this is a hard requirement of go-redis v9's API, unlike older versions.

## Common Pitfalls

- **`BLPop` returning `redis.Nil`.** This isn't a real error — it means the timeout elapsed with nothing to pop. `enqueueAndWait` and `runEngine` both handle it differently: the former treats it as a real timeout error to report; the latter (in the long-running worker loop) just `continue`s and blocks again.
- **Sharing one result key across concurrent requests for the same engine.** The unique-per-job-ID result key (`queue:processed:<engine>:<jobID>`) is what prevents two concurrent requests to the same engine from reading each other's results — a single shared result key per engine would race.
- **Using the request's context for the long-running worker loop.** `runEngine` deliberately uses `context.Background()`, not a request-scoped context — it must keep running after any individual HTTP request completes, unlike `enqueueAndWait`, which correctly uses the request's context so it's cancelled if the client disconnects.
- **Forgetting Redis must be running.** Without it, `main` exits immediately at the `Ping` check — there's no silent degraded mode.

## References

- [go-redis GitHub repository](https://github.com/redis/go-redis)
- [Redis — Lists (RPUSH/BLPOP)](https://redis.io/docs/latest/develop/data-types/lists/)
- [k6 — Get started](https://k6.io/docs/get-started/)

## Next Steps

- [gin](../gin/) — the HTTP framework used here, on its own
- [concurrency/worker-pool](../concurrency/worker-pool/) — the same producer/consumer shape using Go channels instead of Redis
- [mysql](../mysql/) — another example needing a real backing service, for a relational-database comparison
