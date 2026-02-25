package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func decodeBody(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON value")
	}
	return nil
}

func parseIntDefault(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

func readIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host := strings.TrimSpace(r.RemoteAddr)
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		return host[:idx]
	}
	return host
}

func writeMappedError(ctx context.Context, w http.ResponseWriter, operation string, err error) {
	status, code, msg := mapDomainError(err)
	logHTTPOperationError(ctx, operation, status, code, msg, err)
	writeError(w, status, code, msg)
}

func writeValidationError(ctx context.Context, w http.ResponseWriter, operation string, err error) {
	code := "VALIDATION_ERROR"
	msg := err.Error()
	logHTTPOperationError(ctx, operation, http.StatusBadRequest, code, msg, err)
	writeError(w, http.StatusBadRequest, code, msg)
}

func writeMissingBearerError(ctx context.Context, w http.ResponseWriter, operation string) {
	code := "UNAUTHORIZED"
	msg := "missing bearer token"
	logHTTPOperationError(ctx, operation, http.StatusUnauthorized, code, msg, nil)
	writeError(w, http.StatusUnauthorized, code, msg)
}
