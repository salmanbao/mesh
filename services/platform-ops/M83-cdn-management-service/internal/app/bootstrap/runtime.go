package bootstrap

import (
	"context"
	"fmt"
	stdhttp "net/http"
	"time"

	httpadapter "github.com/viralforge/mesh/services/platform-ops/M83-cdn-management-service/internal/adapters/http"
	"github.com/viralforge/mesh/services/platform-ops/M83-cdn-management-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/platform-ops/M83-cdn-management-service/internal/application"
)

type Runtime struct{ httpServer *stdhttp.Server }

func NewRuntime(_ context.Context, configPath string) (*Runtime, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{Config: application.Config{ServiceName: cfg.ServiceID, Version: cfg.Version, IdempotencyTTL: cfg.IdempotencyTTL}, Configs: repos.Configs, Purges: repos.Purges, Metrics: repos.Metrics, Certificates: repos.Certificates, Idempotency: repos.Idempotency})
	server := &stdhttp.Server{Addr: fmt.Sprintf(":%d", cfg.HTTPPort), Handler: httpadapter.NewRouter(httpadapter.NewHandler(svc)), ReadHeaderTimeout: 5 * time.Second}
	return &Runtime{httpServer: server}, nil
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
		if err != nil {
			return err
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.httpServer.Shutdown(shutdownCtx)
}
