package http

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M18-cache-state-management/internal/application"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	actorKey     contextKey = "actor"
)

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-Id"))
		if requestID == "" {
			requestID = "req-" + time.Now().UTC().Format("20060102150405.000000000")
		}
		w.Header().Set("X-Request-Id", requestID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, requestID)))
	})
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token", requestIDFromContext(r.Context()))
			return
		}
		subject := strings.TrimSpace(auth[7:])
		if subject == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "empty bearer token", requestIDFromContext(r.Context()))
			return
		}
		role := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Actor-Role")))
		if role == "" {
			role = "service"
		}
		actor := application.Actor{SubjectID: subject, Role: role, RequestID: requestIDFromContext(r.Context()), IdempotencyKey: strings.TrimSpace(r.Header.Get("Idempotency-Key"))}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorKey, actor)))
	})
}

func metricsMiddleware(service *application.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			if service != nil {
				service.RecordHTTPMetric(r.Context(), r.Method, r.URL.Path, rec.status, time.Since(start))
			}
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) { r.status = code; r.ResponseWriter.WriteHeader(code) }

func actorFromContext(ctx context.Context) application.Actor {
	if v := ctx.Value(actorKey); v != nil {
		if a, ok := v.(application.Actor); ok {
			return a
		}
	}
	return application.Actor{}
}

func requestIDFromContext(ctx context.Context) string {
	if v := ctx.Value(requestIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
