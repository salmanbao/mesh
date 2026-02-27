package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/platform-ops/M67-event-bus/internal/application"
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
		r.Post("/api/v1/events/publish", handler.publishEvent)
		r.Post("/api/v1/topics", handler.createTopic)
		r.Get("/api/v1/topics", handler.listTopics)
		r.Post("/api/v1/acls", handler.createACL)
		r.Get("/api/v1/acls", handler.listACLs)
		r.Post("/api/v1/schemas/register", handler.registerSchema)
		r.Post("/api/v1/consumer-groups/{group_id}/offsets", handler.resetConsumerOffset)
		r.Get("/api/v1/admin/dlq", handler.listDLQ)
		r.Post("/api/v1/admin/dlq/replay", handler.replayDLQ)
	})
	return r
}
