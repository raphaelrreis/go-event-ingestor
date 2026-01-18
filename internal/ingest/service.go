package ingest

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/raphaelreis/go-event-ingestor/internal/metrics"
	"github.com/raphaelreis/go-event-ingestor/internal/model"
	"github.com/raphaelreis/go-event-ingestor/internal/kafka"
)

var (
	ErrQueueFull = errors.New("ingestion queue is full")
)

// Service handles the ingestion flow.
type Service struct {
	queue   chan model.Event
	producer kafka.Producer
	logger  *slog.Logger
	metrics *metrics.Metrics
	wg      sync.WaitGroup
	quit    chan struct{}
}

// NewService creates a new ingestion service with a worker pool.
func NewService(queueSize int, workerCount int, producer kafka.Producer, logger *slog.Logger, m *metrics.Metrics) *Service {
	s := &Service{
		queue:    make(chan model.Event, queueSize),
		producer: producer,
		logger:   logger,
		metrics:  m,
		quit:     make(chan struct{}),
	}

	// Start workers
	for i := 0; i < workerCount; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}

	return s
}

// Ingest receives an event and attempts to enqueue it.
// Returns ErrQueueFull if the queue is at capacity (Backpressure).
func (s *Service) Ingest(ctx context.Context, event model.Event) error {
	select {
	case s.queue <- event:
		s.metrics.IngestQueueSize.Inc()
		s.metrics.EventsReceived.Inc()
		return nil
	default:
		// Queue is full, drop request immediately (Fail Fast)
		return ErrQueueFull
	}
}

func (s *Service) worker(id int) {
	defer s.wg.Done()
	s.logger.Debug("Worker started", "worker_id", id)

	for {
		select {
		case <-s.quit:
			return
		case event := <-s.queue:
			s.metrics.IngestQueueSize.Dec()
			start := time.Now()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := s.producer.Publish(ctx, event)
			cancel()

			duration := time.Since(start).Seconds() * 1000
			s.metrics.IngestLatency.Observe(duration)

			if err != nil {
				// Publish failed (and DLQ failed potentially, or logic in Producer handled it)
				// If Producer.Publish returns error, it means even DLQ failed or critical error
				s.logger.Error("Failed to process event", "event_id", event.ID, "error", err)
				s.metrics.EventsFailed.Inc()
			} else {
				s.metrics.EventsPublished.Inc()
			}
		}
	}
}

// Shutdown gracefully stops the service and waits for workers to finish.
func (s *Service) Shutdown() {
	close(s.quit)
	s.wg.Wait()
	close(s.queue)
}
