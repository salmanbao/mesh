package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/application"
)

type contextKey string

const actorKey contextKey = "actor"

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-Id"))
		if requestID == "" {
			requestID = uuid.NewString()
		}
		ctx := context.WithValue(r.Context(), contextKey("request_id"), requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token", requestIDFromContext(r.Context()))
			return
		}
		subject := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if subject == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "empty bearer token", requestIDFromContext(r.Context()))
			return
		}
		role := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Actor-Role")))
		if role == "" {
			role = "user"
		}
		ctx := context.WithValue(r.Context(), actorKey, application.Actor{
			SubjectID:      subject,
			Role:           role,
			RequestID:      requestIDFromContext(r.Context()),
			IdempotencyKey: strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func actorFromContext(ctx context.Context) application.Actor {
	if value := ctx.Value(actorKey); value != nil {
		if actor, ok := value.(application.Actor); ok {
			return actor
		}
	}
	return application.Actor{}
}

func requestIDFromContext(ctx context.Context) string {
	if value := ctx.Value(contextKey("request_id")); value != nil {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}
	return ""
}
