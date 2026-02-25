package events

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/application"
)

type Message struct {
	Topic   string
	Payload []byte
}

type Consumer interface {
	Poll(ctx context.Context, max int) ([]Message, error)
}

type ConsumerWorker struct {
	logger   *slog.Logger
	consumer Consumer
	service  *application.Service
	interval time.Duration
}

func NewConsumerWorker(logger *slog.Logger, consumer Consumer, service *application.Service, interval time.Duration) *ConsumerWorker {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &ConsumerWorker{
		logger: logger, consumer: consumer, service: service, interval: interval,
	}
}

func (w *ConsumerWorker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		if err := w.processOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			w.logger.ErrorContext(ctx, "consumer iteration failed",
				"module", "events.consumer_worker",
				"layer", "adapter",
				"operation", "process_once",
				"outcome", "failure",
				"error", err,
			)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *ConsumerWorker) processOnce(ctx context.Context) error {
	msgs, err := w.consumer.Poll(ctx, 50)
	if err != nil {
		return err
	}
	for _, msg := range msgs {
		switch msg.Topic {
		case "user.registered":
			if err := w.service.HandleUserRegistered(ctx, msg.Payload); err != nil {
				w.logger.WarnContext(ctx, "failed to handle user.registered", "error", err)
			}
		case "user.deleted":
			if err := w.service.HandleUserDeleted(ctx, msg.Payload); err != nil {
				w.logger.WarnContext(ctx, "failed to handle user.deleted", "error", err)
			}
		default:
			var probe map[string]any
			_ = json.Unmarshal(msg.Payload, &probe)
		}
	}
	return nil
}
