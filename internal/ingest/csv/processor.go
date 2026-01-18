package csv

import (
	"context"
	"encoding/csv"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/raphaelreis/go-event-ingestor/internal/kafka"
	"github.com/raphaelreis/go-event-ingestor/internal/metrics"
	"github.com/raphaelreis/go-event-ingestor/internal/model"
)

type Pipeline struct {
	source   FileSource
	producer kafka.Producer
	logger   *slog.Logger
	metrics  *metrics.Metrics
}

func NewPipeline(source FileSource, producer kafka.Producer, logger *slog.Logger, m *metrics.Metrics) *Pipeline {
	return &Pipeline{
		source:   source,
		producer: producer,
		logger:   logger,
		metrics:  m,
	}
}

func (p *Pipeline) Process(ctx context.Context, cfg Config) error {
	p.logger.Info("Starting bulk CSV ingestion", "file", cfg.FilePath, "workers", cfg.WorkerCount)

	rc, err := p.source.Open(ctx, cfg.FilePath)
	if err != nil {
		return err
	}
	defer rc.Close()

	reader := csv.NewReader(rc)

	jobs := make(chan []string, cfg.BatchSize)

	var wg sync.WaitGroup

	for i := 0; i < cfg.WorkerCount; i++ {
		wg.Add(1)
		go p.worker(ctx, i, jobs, &wg)
	}

	var readErr error
	go func() {
		defer close(jobs)
		lineNum := 0
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				p.logger.Error("CSV parse error", "line", lineNum, "error", err)
				continue
			}
			lineNum++

			select {
			case jobs <- record:
			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Wait()

	if readErr != nil {
		p.logger.Error("Ingestion failed with read errors", "error", readErr)
		return readErr
	}

	p.logger.Info("Bulk ingestion completed successfully", "file", cfg.FilePath)
	return nil
}

func (p *Pipeline) worker(ctx context.Context, id int, jobs <-chan []string, wg *sync.WaitGroup) {
	defer wg.Done()

	for record := range jobs {
		event := model.Event{
			ID:        "csv-row",
			Type:      "bulk_import",
			Timestamp: time.Now(),
			Payload:   map[string]interface{}{"data": record},
		}

		if err := p.producer.Publish(ctx, event); err != nil {
			p.logger.Error("Failed to publish event", "worker", id, "error", err)
		}
	}
}
