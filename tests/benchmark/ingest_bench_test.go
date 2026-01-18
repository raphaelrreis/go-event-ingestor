package benchmark

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/raphaelreis/go-event-ingestor/internal/ingest"
	"github.com/raphaelreis/go-event-ingestor/internal/metrics"
	"github.com/raphaelreis/go-event-ingestor/internal/model"
)

// NoOpProducer implements kafka.Producer but does nothing,
// allowing us to benchmark the Service queuing and worker overhead.
type NoOpProducer struct{}

func (p *NoOpProducer) Publish(ctx context.Context, event model.Event) error {
	return nil
}
func (p *NoOpProducer) Close() error { return nil }

func BenchmarkIngestService(b *testing.B) {
	// Setup
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	mets := metrics.New()
	producer := &NoOpProducer{}
	
	// Create service with ample buffer to test throughput
	svc := ingest.NewService(10000, 10, producer, logger, mets)
	defer svc.Shutdown()

	event := model.Event{
		ID:        "bench-id",
		Type:      "benchmark",
		Timestamp: time.Now(),
		Payload:   map[string]interface{}{"key": "value"},
	}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = svc.Ingest(ctx, event)
	}
}
