package http

import (
	"encoding/json"
	"net/http"

	"github.com/viralforge/mesh/services/integrations/M45-community-service/internal/contracts"
	"github.com/viralforge/mesh/services/integrations/M45-community-service/internal/domain"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
func writeSuccess(w http.ResponseWriter, status int, message string, data any) {
	writeJSON(w, status, contracts.SuccessResponse{Status: "success", Message: message, Data: data})
}
func writeError(w http.ResponseWriter, status int, code, message, requestID string) {
	writeJSON(w, status, contracts.ErrorResponse{Status: "error", Error: contracts.ErrorPayload{Code: code, Message: message, RequestID: requestID}})
}
func mapDomainError(err error) (int, string) {
	switch err {
	case nil:
		return http.StatusOK, ""
	case domain.ErrUnauthorized:
		return http.StatusUnauthorized, "unauthorized"
	case domain.ErrForbidden:
		return http.StatusForbidden, "forbidden"
	case domain.ErrNotFound:
		return http.StatusNotFound, "not_found"
	case domain.ErrInvalidInput:
		return http.StatusBadRequest, "invalid_input"
	case domain.ErrIdempotencyRequired:
		return http.StatusBadRequest, "idempotency_key_required"
	case domain.ErrIdempotencyConflict, domain.ErrConflict:
		return http.StatusConflict, "conflict"
	default:
		return http.StatusInternalServerError, "internal_error"
	}
}
