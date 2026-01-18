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

type Service struct {
	queue    chan model.Event
	producer kafka.Producer
	logger   *slog.Logger
	metrics  *metrics.Metrics
	wg       sync.WaitGroup
}

func NewService(queueSize int, workerCount int, producer kafka.Producer, logger *slog.Logger, m *metrics.Metrics) *Service {
	s := &Service{
		queue:    make(chan model.Event, queueSize),
		producer: producer,
		logger:   logger,
		metrics:  m,
	}

	for i := 0; i < workerCount; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}

	return s
}

func (s *Service) Ingest(ctx context.Context, event model.Event) error {
	select {
	case s.queue <- event:
		s.metrics.IngestQueueSize.Inc()
		s.metrics.EventsReceived.Inc()
		return nil
	default:
		return ErrQueueFull
	}
}

func (s *Service) worker(id int) {
	defer s.wg.Done()
	s.logger.Debug("Worker started", "worker_id", id)

	for event := range s.queue {
		s.metrics.IngestQueueSize.Dec()
		start := time.Now()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := s.producer.Publish(ctx, event)
		cancel()

		duration := time.Since(start).Seconds() * 1000
		s.metrics.IngestLatency.Observe(duration)

		if err != nil {
			s.logger.Error("Failed to process event", "event_id", event.ID, "error", err)
			s.metrics.EventsFailed.Inc()
		} else {
			s.metrics.EventsPublished.Inc()
		}
	}
	s.logger.Debug("Worker stopped", "worker_id", id)
}

func (s *Service) Shutdown() {
	close(s.queue)
	s.wg.Wait()
}