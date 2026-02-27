package http

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/viralforge/mesh/services/integrations/M72-webhook-manager/internal/domain"
)

type successResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type errorResponse struct {
	Status string       `json:"status"`
	Error  errorPayload `json:"error"`
}

type errorPayload struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeSuccess(w http.ResponseWriter, status int, message string, data interface{}) {
	writeJSON(w, status, successResponse{Status: "success", Message: message, Data: data})
}

func writeError(w http.ResponseWriter, status int, code, message, requestID string) {
	writeJSON(w, status, errorResponse{Status: "error", Error: errorPayload{Code: code, Message: message, RequestID: requestID}})
}

func mapDomainError(err error) (int, string) {
	switch err {
	case domain.ErrUnauthorized:
		return http.StatusUnauthorized, "unauthorized"
	case domain.ErrForbidden:
		return http.StatusForbidden, "forbidden"
	case domain.ErrNotFound:
		return http.StatusNotFound, "not_found"
	case domain.ErrInvalidInput:
		return http.StatusBadRequest, "invalid_input"
	case domain.ErrConflict:
		return http.StatusConflict, "conflict"
	case domain.ErrIdempotencyRequired:
		return http.StatusBadRequest, "idempotency_key_required"
	case domain.ErrIdempotencyConflict:
		return http.StatusConflict, "idempotency_conflict"
	default:
		return http.StatusInternalServerError, "internal_error"
	}
}

func ioReadAllLimit(r io.Reader, limit int64) ([]byte, error) {
	lr := io.LimitReader(r, limit)
	return io.ReadAll(lr)
}
