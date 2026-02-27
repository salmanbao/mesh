package bootstrap

import (
	"context"
	"fmt"
	"log"
	stdhttp "net/http"
	"time"

	transporthttp "github.com/viralforge/mesh/services/integrations/M72-webhook-manager/internal/adapters/http"
	"github.com/viralforge/mesh/services/integrations/M72-webhook-manager/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/integrations/M72-webhook-manager/internal/application"
)

type Runtime struct {
	httpServer *stdhttp.Server
}

func NewRuntime(ctx context.Context, configPath string) (*Runtime, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Config: application.Config{
			ServiceName:    cfg.ServiceID,
			Version:        cfg.Version,
			IdempotencyTTL: cfg.IdempotencyTTL,
		},
		Webhooks:    repos.Webhooks,
		Deliveries:  repos.Deliveries,
		Analytics:   repos.Analytics,
		Idempotency: repos.Idempotency,
	})
	handler := transporthttp.NewHandler(svc)
	router := transporthttp.NewRouter(handler)
	s := &stdhttp.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return &Runtime{httpServer: s}, nil
}

func (r *Runtime) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := r.httpServer.ListenAndServe(); err != nil && err != stdhttp.ErrServerClosed {
			errCh <- err
		}
	}()
	select {
	case <-ctx.Done():
	case err := <-errCh:
		log.Printf("runtime error: %v", err)
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.httpServer.Shutdown(shutdownCtx)
}
