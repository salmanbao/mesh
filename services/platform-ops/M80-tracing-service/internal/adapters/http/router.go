package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/platform-ops/M80-tracing-service/internal/application"
)

func NewRouter(handler *Handler, service *application.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(requestIDMiddleware)
	r.Use(metricsMiddleware(service))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeSuccess(w, http.StatusOK, "", map[string]string{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeSuccess(w, http.StatusOK, "", map[string]string{"status": "ready"})
	})
	r.Get("/health", handler.getHealth)
	r.Get("/metrics", handler.getMetrics)

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/ingest/otlp", handler.ingestOTLP)
		r.Post("/ingest/zipkin", handler.ingestZipkin)
		r.Get("/traces", handler.searchTraces)
		r.Get("/traces/{trace_id}", handler.getTrace)
		r.Get("/sampling-policies", handler.listSamplingPolicies)
		r.Post("/sampling-policies", handler.createSamplingPolicy)
		r.Post("/exports", handler.createExport)
		r.Get("/exports/{export_id}", handler.getExport)
	})
	return r
}
