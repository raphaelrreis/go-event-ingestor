//go:build integration

package integration

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/raphaelreis/go-event-ingestor/internal/kafka"
	"github.com/raphaelreis/go-event-ingestor/internal/model"
	kafkaGo "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcKafka "github.com/testcontainers/testcontainers-go/modules/kafka"
)

func TestKafkaIntegration(t *testing.T) {
	ctx := context.Background()

	kafkaContainer, err := tcKafka.Run(ctx,
		"confluentinc/cp-kafka:7.6.1",
		tcKafka.WithClusterID("test-cluster"),
	)
	require.NoError(t, err)
	defer func() {
		if err := kafkaContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}()

	brokers, err := kafkaContainer.Brokers(ctx)
	require.NoError(t, err)
	t.Logf("Kafka started at: %v", brokers)

	topic := "test-events"
	dlqTopic := "test-events-dlq"

	// Create topics explicitly to ensure they exist
	conn, err := kafkaGo.Dial("tcp", brokers[0])
	require.NoError(t, err)
	defer conn.Close()

	controller, err := conn.Controller()
	require.NoError(t, err)
	var controllerConn *kafkaGo.Conn
	controllerConn, err = kafkaGo.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	require.NoError(t, err)
	defer controllerConn.Close()

	topicConfigs := []kafkaGo.TopicConfig{
		{Topic: topic, NumPartitions: 1, ReplicationFactor: 1},
		{Topic: dlqTopic, NumPartitions: 1, ReplicationFactor: 1},
	}

	err = controllerConn.CreateTopics(topicConfigs...)
	require.NoError(t, err)

	producer := kafka.NewProducer(
		brokers,
		topic,
		dlqTopic,
		5*time.Second,
	)
	defer producer.Close()

	event := model.Event{
		ID:        "evt-123",
		Type:      "test-type",
		Timestamp: time.Now(),
		Payload:   map[string]interface{}{"foo": "bar"},
	}

	err = producer.Publish(ctx, event)
	assert.NoError(t, err)

	reader := kafkaGo.NewReader(kafkaGo.ReaderConfig{
		Brokers:   brokers,
		Topic:     topic,
		Partition: 0,
		MaxBytes:  10e6,
	})
	defer reader.Close()

	ctxRead, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	m, err := reader.ReadMessage(ctxRead)
	assert.NoError(t, err)

	assert.Equal(t, event.ID, string(m.Key))
	assert.Contains(t, string(m.Value), "test-type")
	assert.Contains(t, string(m.Value), "foo")
	assert.Contains(t, string(m.Value), "bar")
}