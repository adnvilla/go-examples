# Metric

**Category:** observability
**Difficulty:** Beginner

## Objective

Show sending a counter metric to StatsD/Datadog via `github.com/DataDog/datadog-go/statsd` — the client-side half of a metrics pipeline, independent of whether anything is actually listening.

## Concepts Covered

- `statsd.New(addr)` — constructs a UDP client; since UDP is connectionless, this succeeds even if nothing is listening at `addr`
- `statsd.Count(name, value, tags, rate)` — increments a counter metric, with tags for dimensional filtering (`service:go-examples` here) and a sample rate
- Why sending metrics over UDP is fire-and-forget: a missing or unreachable collector doesn't produce an error, since there's no delivery acknowledgment at the protocol level

## Prerequisites

- Go 1.24+
- Optional: a StatsD-compatible listener on `127.0.0.1:8125` to actually see the metrics land somewhere (this repo's `docker-compose.yml` provides one):
  ```bash
  docker compose up -d statsd
  ```
  Without it, the program still runs — the metric packets are just silently dropped since UDP has no delivery confirmation.

## Project Structure

```
metric/
├── go.mod
├── main.go
└── README.md
```

## How to Run

```bash
make run
# or
go run .
```

Runs forever, sending one `example_metric.histogram` count every second — stop it with `Ctrl+C`.

## Expected Output

```
2026/07/05 17:13:27 Done...
2026/07/05 17:13:28 Done...
2026/07/05 17:13:29 Done...
```

(Identical whether or not a StatsD listener is actually running at `127.0.0.1:8125` — the client has no way to know.)

## Code Walkthrough

- `statsd.New("127.0.0.1:8125")` creates the client. It doesn't perform any network handshake — UDP is connectionless, so "connecting" is really just recording the destination address.
- `statsd.Count("example_metric.histogram", 1, tags, 1)` sends a counter increment: the metric name, the increment value (`1`), a set of tags (`service:go-examples`), and a sample rate (`1` = 100%, i.e. send every call — a lower value like `0.1` would randomly sample only 10% of calls, useful for high-volume metrics).
- The loop runs forever, sleeping one second between sends, logging `"Done..."` regardless of whether the send actually reached a collector.
- `err := statsd.Count(...)` is checked and logged if non-nil — but a `nil` error here only means the client successfully *handed the packet to the OS* for sending, not that any collector received it.

## Common Pitfalls

- **Treating a `nil` error from `Count` as delivery confirmation.** UDP has no acknowledgment; a `nil` error just means the local send call succeeded, not that anything downstream received the metric.
- **Using sample rate `1` for very high-frequency metrics in production.** At high call volumes, sampling (e.g. `0.1`) reduces network/collector load — the client-side library scales the reported value accordingly so aggregate counts remain statistically correct.
- **Forgetting tags entirely.** Without tags like `service:go-examples`, metrics from different services/environments become indistinguishable once aggregated centrally.
- **Assuming this needs the docker-compose `statsd` service to run at all.** It doesn't — the point of this example is precisely that the client behaves identically whether or not a listener exists; only if you want to *see* the metrics land somewhere does the service matter.

## References

- [DataDog/datadog-go GitHub repository](https://github.com/DataDog/datadog-go)
- [StatsD protocol](https://github.com/statsd/statsd/blob/master/docs/metric_types.md)
- [Datadog — DogStatsD](https://docs.datadoghq.com/developers/dogstatsd/)

## Next Steps

- [slog](../slog/) — structured logging, the complementary observability signal to metrics
- [profiling](../profiling/) — CPU profiling, another way of measuring what a program is doing
