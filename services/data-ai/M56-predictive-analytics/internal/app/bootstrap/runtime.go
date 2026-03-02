package bootstrap

import (
	"net/http"
	"strings"

	httpadapter "github.com/viralforge/mesh/services/data-ai/M56-predictive-analytics/internal/adapters/http"
	"github.com/viralforge/mesh/services/data-ai/M56-predictive-analytics/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/data-ai/M56-predictive-analytics/internal/application"
)

type Runtime struct {
	Router http.Handler
	Addr   string
}

func NewRuntime(cfg Config) *Runtime {
	repos := postgres.NewRepositories()
	service := application.NewService(application.Dependencies{
		Config: application.Config{
			ServiceName: "M56-Predictive-Analytics",
		},
		Idempotency: repos.Idempotency,
		Predictions: repos.Predictions,
	})
	handler := httpadapter.NewHandler(service)
	return &Runtime{
		Router: httpadapter.NewRouter(handler),
		Addr:   normalizeAddr(cfg.HTTPPort),
	}
}

func normalizeAddr(port string) string {
	port = strings.TrimSpace(port)
	if port == "" {
		return ":8080"
	}
	if strings.HasPrefix(port, ":") {
		return port
	}
	return ":" + port
}
