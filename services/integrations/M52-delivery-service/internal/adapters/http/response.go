package http

import (
	"encoding/json"
	"net/http"

	"github.com/viralforge/mesh/services/integrations/M52-delivery-service/internal/contracts"
	"github.com/viralforge/mesh/services/integrations/M52-delivery-service/internal/domain"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
func writeSuccess(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, contracts.SuccessResponse{Status: "success", Data: data})
}
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, contracts.ErrorResponse{Status: "error", Code: code, Message: message})
}
func mapDomainError(err error) (int, string) {
	switch err {
	case nil:
		return http.StatusOK, ""
	case domain.ErrUnauthorized:
		return http.StatusUnauthorized, "UNAUTHORIZED"
	case domain.ErrForbidden:
		return http.StatusForbidden, "FORBIDDEN"
	case domain.ErrNotFound:
		return http.StatusNotFound, "NOT_FOUND"
	case domain.ErrInvalidInput:
		return http.StatusBadRequest, "INVALID_INPUT"
	case domain.ErrIdempotencyRequired:
		return http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED"
	case domain.ErrIdempotencyConflict, domain.ErrConflict:
		return http.StatusConflict, "CONFLICT"
	case domain.ErrTokenExpired:
		return http.StatusNotFound, "TOKEN_EXPIRED"
	case domain.ErrAccessRevoked:
		return http.StatusForbidden, "ACCESS_DENIED"
	case domain.ErrDownloadLimitReached:
		return http.StatusForbidden, "DOWNLOAD_LIMIT_REACHED"
	case domain.ErrRateLimited:
		return http.StatusTooManyRequests, "RATE_LIMITED"
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR"
	}
}
