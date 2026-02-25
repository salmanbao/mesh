package http

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/ports"
)

type ctxKey string

const (
	ctxKeyRequestID ctxKey = "request_id"
	ctxKeyClaims    ctxKey = "claims"
	ctxKeyTokenRaw  ctxKey = "token_raw"
)

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-Id")
		if reqID == "" {
			reqID = uuid.NewString()
		}
		w.Header().Set("X-Request-Id", reqID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKeyRequestID, reqID)))
	})
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		_ = start
	})
}

func bearerTokenFromHeader(header string) (string, error) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", domain.ErrUnauthorized
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", domain.ErrUnauthorized
	}
	return token, nil
}

func claimsFromContext(ctx context.Context) (ports.AuthClaims, bool) {
	v := ctx.Value(ctxKeyClaims)
	claims, ok := v.(ports.AuthClaims)
	return claims, ok
}

func requestIDFromContext(ctx context.Context) string {
	v := ctx.Value(ctxKeyRequestID)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func mapDomainError(err error) (int, string, string) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		return http.StatusBadRequest, "VALIDATION_ERROR", err.Error()
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials"
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, "FORBIDDEN", err.Error()
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "NOT_FOUND", "resource not found"
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrIdempotencyConflict):
		return http.StatusConflict, "CONFLICT", err.Error()
	case errors.Is(err, domain.ErrRateLimitExceeded):
		return http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", err.Error()
	case errors.Is(err, domain.ErrDependencyUnavailable), errors.Is(err, domain.ErrStorageUnavailable):
		return http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "service unavailable"
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error"
	}
}
