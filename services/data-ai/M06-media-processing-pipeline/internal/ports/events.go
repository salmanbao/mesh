package ports

import (
	"context"

	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/contracts"
)

type JobQueue interface {
	Enqueue(ctx context.Context, jobID string) error
	Dequeue(ctx context.Context) (string, error)
}

type DLQPublisher interface {
	Publish(ctx context.Context, record contracts.QueueDLQRecord) error
}
