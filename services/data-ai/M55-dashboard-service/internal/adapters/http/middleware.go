package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/application"
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
		if isMutatingMethod(r.Method) && strings.TrimSpace(r.Header.Get("Idempotency-Key")) == "" {
			writeError(w, http.StatusBadRequest, "missing_idempotency_key", "Idempotency-Key is required for mutating operations", requestIDFromContext(r.Context()))
			return
		}
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
			role = "creator"
		}
		actor := application.Actor{
			SubjectID:      subject,
			Role:           role,
			RequestID:      requestIDFromContext(r.Context()),
			IdempotencyKey: strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		}
		ctx := context.WithValue(r.Context(), actorKey, actor)
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

// isMutatingMethod returns true for HTTP verbs that mutate state.
func isMutatingMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
