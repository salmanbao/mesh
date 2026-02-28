package main

import (
	"log/slog"
	"net/http"
	"os"

	httpadapter "github.com/viralforge/mesh/services/trust-compliance/M51-data-portability-service/internal/adapters/http"
	"github.com/viralforge/mesh/services/trust-compliance/M51-data-portability-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/trust-compliance/M51-data-portability-service/internal/application"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger.With("service", "M51-Data-Portability-Service"))

	repos := postgres.NewRepositories()
	service := application.NewService(application.Dependencies{
		Exports:     repos.ExportRequests,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})
	router := httpadapter.NewRouter(httpadapter.NewHandler(service))

	slog.Info("starting api server", "addr", ":8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		slog.Error("api server exited", "error", err)
		os.Exit(1)
	}
}
