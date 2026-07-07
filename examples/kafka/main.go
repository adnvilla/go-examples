// Demonstrates Kafka from Go with segmentio/kafka-go: explicit topic
// creation, a keyed producer (same key -> same partition, which is Kafka's
// per-key ordering guarantee), a consumer group doing at-least-once
// processing (fetch, process, then commit), and offset persistence — a
// "restarted" consumer in the same group resumes exactly where the committed
// offsets left off.
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
)

const broker = "localhost:9092"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Unique topic and group per run: Kafka retains messages and committed
	// offsets, so reusing names would make reruns see leftover state.
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	topic := "examples.orders." + suffix
	group := "examples.group." + suffix

	if err := createTopic(topic, 3); err != nil {
		return err
	}
	fmt.Println("--- topic created with 3 partitions ---")

	if err := produce(ctx, topic); err != nil {
		return err
	}
	if err := consumeGroup(ctx, topic, group, 6); err != nil {
		return err
	}
	return resumeAfterRestart(ctx, topic, group)
}

// createTopic creates the topic explicitly (3 partitions, RF 1) against the
// cluster controller — relying on auto-creation hands you the broker's
// default partition count and hides typos as fresh empty topics.
func createTopic(topic string, partitions int) error {
	conn, err := kafka.Dial("tcp", broker)
	if err != nil {
		return fmt.Errorf("kafka is not reachable on %s (docker compose up -d kafka): %w", broker, err)
	}
	defer conn.Close() //nolint:errcheck

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("finding controller: %w", err)
	}
	controllerConn, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return fmt.Errorf("dialing controller: %w", err)
	}
	defer controllerConn.Close() //nolint:errcheck

	err = controllerConn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     partitions,
		ReplicationFactor: 1,
	})
	if err != nil {
		return fmt.Errorf("creating topic: %w", err)
	}

	// Topic creation is asynchronous: metadata takes a moment to propagate,
	// and producing before that fails with Unknown Topic Or Partition. Poll
	// until the broker reports all partitions.
	deadline := time.Now().Add(15 * time.Second)
	for {
		parts, err := conn.ReadPartitions(topic)
		if err == nil && len(parts) == partitions {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("topic %s not visible after creation (last: %d/%d partitions, err=%v)",
				topic, len(parts), partitions, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// produce writes six keyed messages. The Hash balancer routes every message
// with the same key to the same partition — that partition affinity is the
// entire basis of Kafka's ordering model: order is guaranteed per partition,
// therefore per key, and nowhere else.
func produce(ctx context.Context, topic string) error {
	fmt.Println("\n--- producer: six keyed messages, Hash balancer ---")
	writer := &kafka.Writer{
		Addr:         kafka.TCP(broker),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll, // don't report success until the broker has it
	}
	defer writer.Close() //nolint:errcheck

	var messages []kafka.Message
	for i := 1; i <= 6; i++ {
		user := []string{"user-a", "user-b", "user-c"}[(i-1)%3]
		messages = append(messages, kafka.Message{
			Key:   []byte(user),
			Value: []byte(fmt.Sprintf("order-%d", i)),
		})
	}
	if err := writeMessages(ctx, writer, messages...); err != nil {
		return fmt.Errorf("writing messages: %w", err)
	}
	fmt.Println("producer: 6 messages acknowledged by the broker")
	return nil
}

// consumeGroup reads as part of a consumer group with the at-least-once
// discipline: fetch, process, then commit. Committing before processing
// would flip the guarantee to at-most-once (a crash loses the message).
func consumeGroup(ctx context.Context, topic, group string, expect int) error {
	fmt.Println("\n--- consumer group: fetch -> process -> commit ---")
	reader := newGroupReader(topic, group)
	defer reader.Close() //nolint:errcheck

	byKey := map[string][]string{}
	partitionOf := map[string]int{}
	for range expect {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			return fmt.Errorf("fetching: %w", err)
		}
		key := string(msg.Key)
		byKey[key] = append(byKey[key], string(msg.Value))
		if p, seen := partitionOf[key]; seen && p != msg.Partition {
			return fmt.Errorf("key %s seen on partitions %d and %d — hash balancing broken", key, p, msg.Partition)
		}
		partitionOf[key] = msg.Partition
		// Processing happened (we recorded it) — now it's safe to commit.
		if err := reader.CommitMessages(ctx, msg); err != nil {
			return fmt.Errorf("committing: %w", err)
		}
	}

	keys := make([]string, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("%s: %v (single partition, in order)\n", k, byKey[k])
	}
	return nil
}

// resumeAfterRestart simulates a consumer restart: a NEW reader in the SAME
// group starts from the committed offsets, so it sees only what was produced
// after the previous consumer stopped — nothing is reprocessed, nothing lost.
func resumeAfterRestart(ctx context.Context, topic, group string) error {
	fmt.Println("\n--- restart: same group resumes from committed offsets ---")

	writer := &kafka.Writer{Addr: kafka.TCP(broker), Topic: topic, Balancer: &kafka.Hash{}, RequiredAcks: kafka.RequireAll}
	if err := writeMessages(ctx, writer, kafka.Message{Key: []byte("user-a"), Value: []byte("order-7")}); err != nil {
		return fmt.Errorf("writing post-restart message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("closing writer: %w", err)
	}

	reader := newGroupReader(topic, group)
	defer reader.Close() //nolint:errcheck

	msg, err := reader.FetchMessage(ctx)
	if err != nil {
		return fmt.Errorf("fetching after restart: %w", err)
	}
	if err := reader.CommitMessages(ctx, msg); err != nil {
		return fmt.Errorf("committing after restart: %w", err)
	}
	fmt.Printf("restarted consumer received only the new message: %s=%s\n", msg.Key, msg.Value)
	return nil
}

// writeMessages produces with a bounded retry on UnknownTopicOrPartition:
// topic creation is asynchronous, and for a moment after CreateTopics the
// partition leaders aren't ready to accept writes yet.
func writeMessages(ctx context.Context, writer *kafka.Writer, msgs ...kafka.Message) error {
	deadline := time.Now().Add(15 * time.Second)
	for {
		err := writer.WriteMessages(ctx, msgs...)
		if err == nil {
			return nil
		}
		if !isUnknownTopic(err) || time.Now().After(deadline) {
			return err
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func isUnknownTopic(err error) bool {
	if errors.Is(err, kafka.UnknownTopicOrPartition) {
		return true
	}
	var writeErrs kafka.WriteErrors
	if errors.As(err, &writeErrs) {
		for _, e := range writeErrs {
			if errors.Is(e, kafka.UnknownTopicOrPartition) {
				return true
			}
		}
	}
	return false
}

func newGroupReader(topic, group string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{broker},
		Topic:   topic,
		GroupID: group,
		MaxWait: 250 * time.Millisecond,
	})
}
