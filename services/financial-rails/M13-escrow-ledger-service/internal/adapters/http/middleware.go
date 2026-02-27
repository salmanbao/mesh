package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/financial-rails/M13-escrow-ledger-service/internal/application"
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
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token", requestIDFromContext(r.Context()))
			return
		}
		sub := strings.TrimSpace(auth[7:])
		if sub == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "empty bearer token", requestIDFromContext(r.Context()))
			return
		}
		role := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Actor-Role")))
		if role == "" {
			role = "analyst"
		}
		actor := application.Actor{SubjectID: sub, Role: role, RequestID: requestIDFromContext(r.Context()), IdempotencyKey: strings.TrimSpace(r.Header.Get("Idempotency-Key"))}
		ctx := context.WithValue(r.Context(), actorKey, actor)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func actorFromContext(ctx context.Context) application.Actor {
	if v := ctx.Value(actorKey); v != nil {
		if a, ok := v.(application.Actor); ok {
			return a
		}
	}
	return application.Actor{}
}
func requestIDFromContext(ctx context.Context) string {
	if v := ctx.Value(contextKey("request_id")); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
