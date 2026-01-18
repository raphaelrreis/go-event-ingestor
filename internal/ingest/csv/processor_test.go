package csv_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/raphaelreis/go-event-ingestor/internal/ingest/csv"
	"github.com/raphaelreis/go-event-ingestor/internal/metrics"
	"github.com/raphaelreis/go-event-ingestor/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockFileSource struct {
	mock.Mock
}

func (m *MockFileSource) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	args := m.Called(ctx, path)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockFileSource) Checkpoint(ctx context.Context, path string, offset int64) error {
	args := m.Called(ctx, path, offset)
	return args.Error(0)
}

func (m *MockFileSource) ResumeOffset(ctx context.Context, path string) (int64, error) {
	args := m.Called(ctx, path)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockFileSource) MarkCompleted(ctx context.Context, path string) error {
	args := m.Called(ctx, path)
	return args.Error(0)
}

type MockProducer struct {
	mock.Mock
}

func (m *MockProducer) Publish(ctx context.Context, event model.Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockProducer) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestPipeline_Process(t *testing.T) {
	mockSource := new(MockFileSource)
	mockProducer := new(MockProducer)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	mets := metrics.New()

	pipeline := csv.NewPipeline(mockSource, mockProducer, logger, mets)

	csvContent := "col1,col2\nval1,val2\nval3,val4"
	rc := io.NopCloser(strings.NewReader(csvContent))

	mockSource.On("Open", mock.Anything, "test.csv").Return(rc, nil)

	mockProducer.On("Publish", mock.Anything, mock.MatchedBy(func(e model.Event) bool {
		return e.Type == "bulk_import"
	})).Return(nil).Times(3)

	cfg := csv.Config{
		FilePath:    "test.csv",
		WorkerCount: 1,
		BatchSize:   10,
	}

	err := pipeline.Process(context.Background(), cfg)

	assert.NoError(t, err)
	mockSource.AssertExpectations(t)
	mockProducer.AssertExpectations(t)
}
