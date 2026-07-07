package main

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

// requireLocalKafka skips unless KAFKA_LOCAL=1 (same convention as the other
// infra-backed examples) and returns a unique topic for the test.
func requireLocalKafka(t *testing.T) string {
	t.Helper()
	if os.Getenv("KAFKA_LOCAL") == "" {
		t.Skip("set KAFKA_LOCAL=1 to run Kafka integration tests (requires local Kafka on :9092)")
	}
	topic := "examples.test." + t.Name() + "." + strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := createTopic(topic, 3); err != nil {
		t.Fatalf("creating topic: %v", err)
	}
	return topic
}

func TestProduceConsumeRoundTrip(t *testing.T) {
	t.Parallel()
	topic := requireLocalKafka(t)

	writer := &kafka.Writer{Addr: kafka.TCP(broker), Topic: topic, Balancer: &kafka.Hash{}, RequiredAcks: kafka.RequireAll}
	want := map[string]bool{"m1": true, "m2": true, "m3": true}
	var msgs []kafka.Message
	for v := range want {
		msgs = append(msgs, kafka.Message{Key: []byte("k"), Value: []byte(v)})
	}
	if err := writeMessages(t.Context(), writer, msgs...); err != nil {
		t.Fatalf("writing: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("closing writer: %v", err)
	}

	reader := newGroupReader(topic, topic+".group")
	defer reader.Close() //nolint:errcheck

	got := map[string]bool{}
	for range len(want) {
		msg, err := reader.FetchMessage(t.Context())
		if err != nil {
			t.Fatalf("fetching: %v", err)
		}
		got[string(msg.Value)] = true
		if err := reader.CommitMessages(t.Context(), msg); err != nil {
			t.Fatalf("committing: %v", err)
		}
	}
	for v := range want {
		if !got[v] {
			t.Errorf("message %q was produced but never consumed", v)
		}
	}
}

func TestSameKeyLandsOnSamePartition(t *testing.T) {
	t.Parallel()
	topic := requireLocalKafka(t)

	writer := &kafka.Writer{Addr: kafka.TCP(broker), Topic: topic, Balancer: &kafka.Hash{}, RequiredAcks: kafka.RequireAll}
	var msgs []kafka.Message
	for i := range 6 {
		key := []string{"alpha", "beta"}[i%2]
		msgs = append(msgs, kafka.Message{Key: []byte(key), Value: []byte(strconv.Itoa(i))})
	}
	if err := writeMessages(t.Context(), writer, msgs...); err != nil {
		t.Fatalf("writing: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("closing writer: %v", err)
	}

	reader := newGroupReader(topic, topic+".group")
	defer reader.Close() //nolint:errcheck

	partitionOf := map[string]int{}
	for range 6 {
		msg, err := reader.FetchMessage(t.Context())
		if err != nil {
			t.Fatalf("fetching: %v", err)
		}
		key := string(msg.Key)
		if p, seen := partitionOf[key]; seen && p != msg.Partition {
			t.Fatalf("key %q on partitions %d and %d — affinity broken", key, p, msg.Partition)
		}
		partitionOf[key] = msg.Partition
		if err := reader.CommitMessages(t.Context(), msg); err != nil {
			t.Fatalf("committing: %v", err)
		}
	}
}
