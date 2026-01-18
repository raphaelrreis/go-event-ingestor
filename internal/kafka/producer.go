package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/raphaelreis/go-event-ingestor/internal/model"
	"github.com/segmentio/kafka-go"
)

type Producer interface {
	Publish(ctx context.Context, event model.Event) error
	Close() error
}

type KafkaProducer struct {
	writer    *kafka.Writer
	dlqWriter *kafka.Writer
}

func NewProducer(brokers []string, topic, dlqTopic string, timeout time.Duration) *KafkaProducer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: timeout,
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		Async:        false,
		MaxAttempts:  3,
	}

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

func (p *KafkaProducer) Publish(ctx context.Context, event model.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(event.ID),
		Value: payload,
		Headers: []kafka.Header{
			{Key: "trace_id", Value: []byte(event.ID)},
		},
	}

	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
		return p.sendToDLQ(ctx, msg, err)
	}

	return nil
}

func (p *KafkaProducer) sendToDLQ(ctx context.Context, msg kafka.Message, originalErr error) error {
	msg.Headers = append(msg.Headers, kafka.Header{
		Key:   "error",
		Value: []byte(originalErr.Error()),
	})

	if err := p.dlqWriter.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to send to DLQ (original error: %v): %w", originalErr, err)
	}
	return nil
}

func (p *KafkaProducer) Close() error {
	err1 := p.writer.Close()
	err2 := p.dlqWriter.Close()

	if err1 != nil {
		return err1
	}
	return err2
}