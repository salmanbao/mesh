package http

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
)

type ctxKey string

const (
	ctxKeyRequestID ctxKey = "request_id"
	ctxKeyTokenRaw  ctxKey = "token_raw"
	ctxKeyClaims    ctxKey = "auth_claims"
)

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-Id")
		if reqID == "" {
			reqID = uuid.NewString()
		}
		w.Header().Set("X-Request-Id", reqID)
		ctx := context.WithValue(r.Context(), ctxKeyRequestID, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				httpLogger().ErrorContext(r.Context(), "panic recovered",
					"operation", "http_panic_recovery",
					"outcome", "failure",
					"request_id", requestIDFromContext(r.Context()),
					"method", r.Method,
					"path", r.URL.Path,
					"panic", rec,
				)
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Write(payload []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(payload)
	r.bytes += n
	return n, err
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)

		statusCode := recorder.statusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		outcome := "success"
		if statusCode >= 400 {
			outcome = "failure"
		}

		fields := []any{
			"operation", "http_request",
			"outcome", outcome,
			"method", r.Method,
			"path", r.URL.Path,
			"status_code", statusCode,
			"bytes", recorder.bytes,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", requestIDFromContext(r.Context()),
		}
		switch {
		case statusCode >= 500:
			httpLogger().ErrorContext(r.Context(), "http request completed", fields...)
		case statusCode >= 400:
			httpLogger().WarnContext(r.Context(), "http request completed", fields...)
		default:
			httpLogger().InfoContext(r.Context(), "http request completed", fields...)
		}
	})
}

func requestIDFromContext(ctx context.Context) string {
	v := ctx.Value(ctxKeyRequestID)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func bearerTokenFromHeader(header string) (string, error) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", errors.New("missing bearer token")
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", errors.New("missing bearer token")
	}
	return token, nil
}

func mapDomainError(err error) (int, string, string) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		return http.StatusBadRequest, "VALIDATION_ERROR", err.Error()
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials"
	case errors.Is(err, domain.ErrInvalidCredentials):
		return http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password"
	case errors.Is(err, domain.ErrAccountLocked):
		return http.StatusTooManyRequests, "ACCOUNT_LOCKED", "account temporarily locked"
	case errors.Is(err, domain.ErrRateLimited):
		return http.StatusTooManyRequests, "RATE_LIMITED", "too many requests"
	case errors.Is(err, domain.ErrSessionExpired):
		return http.StatusUnauthorized, "SESSION_EXPIRED", "session expired"
	case errors.Is(err, domain.ErrSessionRevoked):
		return http.StatusUnauthorized, "SESSION_REVOKED", "session revoked"
	case errors.Is(err, domain.ErrTokenExpired):
		return http.StatusUnauthorized, "TOKEN_EXPIRED", "token expired"
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrIdempotencyConflict):
		return http.StatusConflict, "CONFLICT", err.Error()
	case errors.Is(err, domain.ErrOIDCFlowRequired):
		return http.StatusBadRequest, "OIDC_FLOW_REQUIRED", "oidc_flow_required"
	case errors.Is(err, domain.ErrCannotUnlinkLastAuth):
		return http.StatusBadRequest, "CANNOT_UNLINK_LAST_METHOD", err.Error()
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "NOT_FOUND", "resource not found"
	case errors.Is(err, domain.ErrNotImplemented):
		return http.StatusNotImplemented, "NOT_IMPLEMENTED", err.Error()
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error"
	}
}

func claimsFromContext(ctx context.Context) (ports.AuthClaims, bool) {
	v := ctx.Value(ctxKeyClaims)
	claims, ok := v.(ports.AuthClaims)
	return claims, ok
}
