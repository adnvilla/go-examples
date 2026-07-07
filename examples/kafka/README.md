# Kafka

**Category:** kafka
**Difficulty:** Advanced

## Objective

Show the Kafka concepts that matter from Go, using `segmentio/kafka-go` against a single-broker KRaft cluster: explicit topic creation, a keyed producer whose Hash balancer gives **same key → same partition** (the entire basis of Kafka's ordering model), a consumer group running the **at-least-once** discipline (fetch → process → commit, in that order), and offset persistence — a "restarted" consumer in the same group resumes exactly where the committed offsets left off, reprocessing nothing.

## Concepts Covered

- Explicit `CreateTopics` against the cluster controller (partitions and replication chosen deliberately, not inherited from broker defaults) — and the fact that creation is *asynchronous*, handled with a bounded produce retry on `UnknownTopicOrPartition`
- `kafka.Writer` with `Balancer: &kafka.Hash{}` and `RequiredAcks: RequireAll` — keys drive partition affinity; acks decide when "sent" means "stored"
- Ordering scope: guaranteed per partition (therefore per key), and nowhere else — the demo asserts each key stays on one partition
- Consumer groups: `FetchMessage` → process → `CommitMessages`, and why committing *before* processing silently flips the guarantee from at-least-once to at-most-once
- Committed offsets as durable group state: a new reader in the same group receives only what arrived after the last commit
- Env-guarded integration tests (`KAFKA_LOCAL=1`) with unique per-test topics

## Prerequisites

- Go 1.25+
- Kafka on `localhost:9092` (this repo's compose provides a single-broker KRaft cluster, no ZooKeeper):
  ```bash
  docker compose up -d kafka   # or, from this directory: make infra-up
  ```
- Dependency justified: `segmentio/kafka-go` is a canonical pure-Go client and Kafka is the topic being taught

## Project Structure

```
kafka/
├── go.mod
├── main.go        # topic creation, producer, consumer group, restart demo
├── main_test.go   # integration tests (KAFKA_LOCAL=1)
├── Makefile
└── README.md
```

## How to Run

```bash
make infra-up   # start Kafka
make run        # the demo (uses a unique topic per run — reruns are clean)
make test       # integration tests (sets KAFKA_LOCAL=1)
```

## Expected Output

```
--- topic created with 3 partitions ---

--- producer: six keyed messages, Hash balancer ---
producer: 6 messages acknowledged by the broker

--- consumer group: fetch -> process -> commit ---
user-a: [order-1 order-4] (single partition, in order)
user-b: [order-2 order-5] (single partition, in order)
user-c: [order-3 order-6] (single partition, in order)

--- restart: same group resumes from committed offsets ---
restarted consumer received only the new message: user-a=order-7
```

## Code Walkthrough

- `createTopic` dials the cluster, locates the controller, and creates the topic with 3 partitions explicitly. Relying on broker auto-creation is the classic footgun: you inherit whatever default partition count the broker has, and a typo in a topic name becomes a fresh empty topic instead of an error. The follow-up poll on `ReadPartitions` plus the produce-side retry (`writeMessages`) absorb the asynchronous window where the topic exists but its partition leaders aren't ready.
- `produce` writes six messages with three keys through the `Hash` balancer. The consumer side *verifies* the resulting affinity rather than asserting it in prose: seeing one key on two partitions fails the run. Per-key ordering (`order-1` before `order-4` for `user-a`) follows directly — Kafka orders within a partition only.
- `consumeGroup` runs the at-least-once loop: `FetchMessage` (which does *not* advance the group's offsets), process, then `CommitMessages`. The order is load-bearing — commit-then-process means a crash between the two steps loses the message; process-then-commit means a crash reprocesses it, which is why Kafka consumers must be idempotent.
- `resumeAfterRestart` closes the reader, produces `order-7`, and opens a *new* reader in the same group. It receives exactly one message: the group's committed offsets — stored broker-side in `__consumer_offsets` — are what survive restarts, not anything in the client.
- Topic and group names carry a per-run suffix: Kafka retains both messages and group offsets, so fixed names would make every rerun see the previous run's leftovers.

## Common Pitfalls

- **Expecting global ordering.** Kafka orders per partition. Consumers reading multiple partitions interleave; if two events must stay ordered, they must share a key.
- **Committing before processing.** The loop shape decides your delivery guarantee. Fetch → commit → process is at-most-once wearing at-least-once's clothes.
- **Assuming exactly-once from at-least-once + commits.** Rebalances and crashes *will* redeliver; handlers need idempotency (or Kafka transactions, a much bigger hammer).
- **Producing immediately after creating a topic.** Creation is async; the first writes can fail with `UnknownTopicOrPartition`. Retry briefly (as here) or create topics ahead of deploment.
- **Keys chosen for uniqueness instead of locality.** A UUID key spreads perfectly but orders nothing. Key by the entity whose events must be sequential (user, order, device).
- **One consumer group name shared by unrelated services.** Group = one logical subscriber; two services sharing a group *split* the messages between them instead of each getting a copy.

## References

- [Apache Kafka documentation — Design](https://kafka.apache.org/documentation/#design)
- [segmentio/kafka-go](https://github.com/segmentio/kafka-go)
- [Confluent — Kafka consumer design: at-least-once semantics](https://docs.confluent.io/kafka/design/delivery-semantics.html)

## Next Steps

- [redis](../redis/) — a lighter-weight queue for when Kafka is too much machinery
- [distributed-lock](../distributed-lock/) — coordination when consumers must not overlap outside partition assignment
- [otel](../otel/) — traces are how produced-to-consumed latency becomes visible
