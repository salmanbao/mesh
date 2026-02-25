package events

import (
	"context"
	"log/slog"
	"time"

	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/application"
)

// OIDCTokenRefreshWorker periodically refreshes provider tokens before expiry.
type OIDCTokenRefreshWorker struct {
	logger    *slog.Logger
	service   *application.Service
	interval  time.Duration
	window    time.Duration
	batchSize int
}

func NewOIDCTokenRefreshWorker(
	logger *slog.Logger,
	service *application.Service,
	interval time.Duration,
	window time.Duration,
	batchSize int,
) *OIDCTokenRefreshWorker {
	if interval <= 0 {
		interval = time.Hour
	}
	if window <= 0 {
		window = 24 * time.Hour
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	return &OIDCTokenRefreshWorker{
		logger:    logger,
		service:   service,
		interval:  interval,
		window:    window,
		batchSize: batchSize,
	}
}

func (w *OIDCTokenRefreshWorker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		if err := w.service.RefreshExpiringOIDCTokens(ctx, w.window, w.batchSize); err != nil {
			w.logger.ErrorContext(ctx, "oidc token refresh iteration failed",
				"module", "events.oidc_refresh_worker",
				"layer", "adapter",
				"operation", "refresh_expiring_oidc_tokens",
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
