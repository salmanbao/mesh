package events

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/application"
)

type Worker struct {
	logger       *slog.Logger
	service      *application.Service
	pollInterval time.Duration
}

func NewWorker(logger *slog.Logger, service *application.Service, pollInterval time.Duration) *Worker {
	return &Worker{logger: logger, service: service, pollInterval: pollInterval}
}

func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.service.ProcessNextJob(ctx); err != nil {
				if errors.Is(err, io.EOF) {
					continue
				}
				w.logger.ErrorContext(ctx, "job processing failed", "error", err)
			}
		}
	}
}
