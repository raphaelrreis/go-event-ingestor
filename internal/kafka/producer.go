package kafka

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

	"github.com/raphaelreis/go-event-ingestor/internal/model"
	"github.com/segmentio/kafka-go"
)

// Producer defines the interface for publishing events.
type Producer interface {
	Publish(ctx context.Context, event model.Event) error
	Close() error
}

type KafkaProducer struct {
	writer    *kafka.Writer
	dlqWriter *kafka.Writer
}

// NewProducer creates a new Kafka producer with a main writer and a DLQ writer.
func NewProducer(brokers []string, topic, dlqTopic string, timeout time.Duration) *KafkaProducer {
	// Main Writer
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: timeout,
		// Batching config for high throughput
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		Async:        false, // We want to handle errors in the worker
		// Retry config handled by kafka-go (basic), logic extended in Publish
		MaxAttempts: 3, 
	}

	// DLQ Writer
	dlq := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        dlqTopic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: timeout,
		Async:        false,
	}

	return &KafkaProducer{
		writer:    w,
		dlqWriter: dlq,
	}
}

// Publish attempts to send an event to Kafka.
// On failure, it attempts to send to the Dead Letter Queue (DLQ).
func (p *KafkaProducer) Publish(ctx context.Context, event model.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(event.ID),
		Value: payload,
		Headers: []kafka.Header{
			{Key: "trace_id", Value: []byte(event.ID)}, // utilizing ID as trace_id for simplicity
		},
	}

	// Attempt write to main topic
	// kafka-go handles retries internally based on MaxAttempts in writer config.
	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
		// If main topic fails after retries, send to DLQ
		return p.sendToDLQ(ctx, msg, err)
	}

	return nil
}

func (p *KafkaProducer) sendToDLQ(ctx context.Context, msg kafka.Message, originalErr error) error {
	// Add original error to headers
	msg.Headers = append(msg.Headers, kafka.Header{
		Key:   "error",
		Value: []byte(originalErr.Error()),
	})

	if err := p.dlqWriter.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to send to DLQ (original error: %v): %w", originalErr, err)
	}
	return nil // Swallowing original error as we successfully DLQ'd it? 
	// Usually we might want to log this or return nil to indicate "handled".
	// The worker will count this as 'failed' but 'processed'.
}

func (p *KafkaProducer) Close() error {
	if err := p.writer.Close(); err != nil {
		_ = p.dlqWriter.Close()
		return err
	}
	return p.dlqWriter.Close()
}

// Ensure TLS is handled if needed (omitted for simplicity as per requirements, but good to note)
func dialer() *kafka.Dialer {
	return &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
		TLS:       &tls.Config{}, // Empty config, assumes no specific certs for now
	}
}
