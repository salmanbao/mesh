package http

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/application"
)

func (h *Handler) listSessions(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenFromContext(r)
	if !ok {
		writeMissingBearerError(r.Context(), w, "list_sessions")
		return
	}
	items, err := h.service.ListSessions(r.Context(), token)
	if err != nil {
		writeMappedError(r.Context(), w, "list_sessions", err)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{"sessions": items})
}

func (h *Handler) revokeSession(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenFromContext(r)
	if !ok {
		writeMissingBearerError(r.Context(), w, "revoke_session")
		return
	}
	sessionID, err := uuid.Parse(chi.URLParam(r, "session_id"))
	if err != nil {
		writeValidationError(r.Context(), w, "revoke_session", errors.New("invalid session_id"))
		return
	}
	if err := h.service.RevokeSessionByID(r.Context(), token, sessionID); err != nil {
		writeMappedError(r.Context(), w, "revoke_session", err)
		return
	}
	writeMessage(w, http.StatusOK, "Session revoked successfully")
}

func (h *Handler) revokeAllSessions(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenFromContext(r)
	if !ok {
		writeMissingBearerError(r.Context(), w, "revoke_all_sessions")
		return
	}
	if err := h.service.LogoutAllSessions(r.Context(), token); err != nil {
		writeMappedError(r.Context(), w, "revoke_all_sessions", err)
		return
	}
	writeMessage(w, http.StatusOK, "All sessions revoked successfully")
}

func (h *Handler) loginHistory(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenFromContext(r)
	if !ok {
		writeMissingBearerError(r.Context(), w, "login_history")
		return
	}

	query := application.LoginHistoryQuery{
		Page:   parseIntDefault(r.URL.Query().Get("page"), 1),
		Limit:  parseIntDefault(r.URL.Query().Get("limit"), 20),
		Days:   parseIntDefault(r.URL.Query().Get("days"), 0),
		Status: r.URL.Query().Get("status"),
	}
	items, err := h.service.ListLoginHistory(r.Context(), token, query)
	if err != nil {
		writeMappedError(r.Context(), w, "login_history", err)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{"attempts": items})
}
