package csv

import (
	"context"
	"io"
)

type FileSource interface {
	Open(ctx context.Context, path string) (io.ReadCloser, error)
	Checkpoint(ctx context.Context, path string, offset int64) error
	ResumeOffset(ctx context.Context, path string) (int64, error)
	MarkCompleted(ctx context.Context, path string) error
}

type Config struct {
	FilePath    string
	WorkerCount int
	BatchSize   int
	RateLimit   float64
}

type Processor interface {
	Process(ctx context.Context, cfg Config) error
}
