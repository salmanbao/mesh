package http

import (
	"context"
	"log/slog"
)

const (
	serviceName = "M01-Authentication-Service"
)

func httpLogger() *slog.Logger {
	return slog.Default().With(
		"service", serviceName,
		"module", "http",
		"layer", "adapter",
	)
}

func logHTTPOperationError(ctx context.Context, operation string, statusCode int, code, message string, err error) {
	fields := []any{
		"operation", operation,
		"outcome", "failure",
		"status_code", statusCode,
		"error_code", code,
		"message", message,
		"request_id", requestIDFromContext(ctx),
	}
	if err != nil {
		fields = append(fields, "error", err.Error())
	}
	if statusCode >= 500 {
		httpLogger().ErrorContext(ctx, "http operation failed", fields...)
		return
	}
	httpLogger().WarnContext(ctx, "http operation failed", fields...)
}
