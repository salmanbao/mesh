package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/integrations/M03-notification-service/internal/application"
	"github.com/viralforge/mesh/services/integrations/M03-notification-service/internal/contracts"
	"github.com/viralforge/mesh/services/integrations/M03-notification-service/internal/domain"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) listNotifications(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	limit, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
	items, nextCursor, unreadCount, err := h.service.ListNotifications(r.Context(), actor, application.ListNotificationsInput{
		Type:   strings.TrimSpace(r.URL.Query().Get("type")),
		Status: strings.TrimSpace(r.URL.Query().Get("status")),
		Limit:  limit,
		Cursor: strings.TrimSpace(r.URL.Query().Get("cursor")),
	})
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	resp := contracts.ListNotificationsResponse{Items: make([]contracts.NotificationItem, 0, len(items)), NextCursor: nextCursor, UnreadCount: unreadCount}
	for _, n := range items {
		resp.Items = append(resp.Items, toNotificationItem(n))
	}
	writeSuccess(w, http.StatusOK, "notifications", resp)
}

func (h *Handler) unreadCount(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	count, err := h.service.UnreadCount(r.Context(), actor)
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "unread count", contracts.UnreadCountResponse{UnreadCount: count})
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	row, err := h.service.MarkRead(r.Context(), actor, chi.URLParam(r, "id"))
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "notification marked read", contracts.MarkStateResponse{NotificationID: row.NotificationID, Status: "read"})
}

func (h *Handler) archive(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	row, err := h.service.Archive(r.Context(), actor, chi.URLParam(r, "id"))
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "notification archived", contracts.MarkStateResponse{NotificationID: row.NotificationID, Status: "archived"})
}

func (h *Handler) bulkAction(w http.ResponseWriter, r *http.Request) {
	var req contracts.BulkActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	actor := actorFromContext(r.Context())
	processed, failed, err := h.service.BulkAction(r.Context(), actor, application.BulkActionInput{Action: req.Action, NotificationIDs: req.NotificationIDs})
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "bulk action applied", contracts.BulkActionResponse{Action: strings.ToLower(strings.TrimSpace(req.Action)), Processed: processed, Failed: failed})
}

func (h *Handler) getPreferences(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	prefs, err := h.service.GetPreferences(r.Context(), actor)
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "notification preferences", toPreferencesResponse(prefs))
}

func (h *Handler) updatePreferences(w http.ResponseWriter, r *http.Request) {
	var req contracts.UpdatePreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	actor := actorFromContext(r.Context())
	prefs, err := h.service.UpdatePreferences(r.Context(), actor, application.UpdatePreferencesInput{
		EmailEnabled: req.EmailEnabled, PushEnabled: req.PushEnabled, SMSEnabled: req.SMSEnabled, InAppEnabled: req.InAppEnabled,
		QuietHoursEnabled: req.QuietHoursEnabled, QuietHoursStart: req.QuietHoursStart, QuietHoursEnd: req.QuietHoursEnd, QuietHoursTZ: req.QuietHoursTZ,
		Language: req.Language, BatchingEnabled: req.BatchingEnabled, MutedTypes: req.MutedTypes,
	})
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "notification preferences updated", toPreferencesResponse(prefs))
}

func (h *Handler) deleteScheduled(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	cancelled, err := h.service.CancelScheduled(r.Context(), actor, chi.URLParam(r, "id"))
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "scheduled notification cancel processed", contracts.DeleteScheduledResponse{ScheduledID: chi.URLParam(r, "id"), Cancelled: cancelled})
}

func toNotificationItem(n domain.Notification) contracts.NotificationItem {
	item := contracts.NotificationItem{NotificationID: n.NotificationID, UserID: n.UserID, Type: n.Type, Title: n.Title, Body: n.Body, Metadata: n.Metadata, CreatedAt: n.CreatedAt.UTC().Format(time.RFC3339)}
	if n.ReadAt != nil {
		item.ReadAt = n.ReadAt.UTC().Format(time.RFC3339)
	}
	if n.ArchivedAt != nil {
		item.ArchivedAt = n.ArchivedAt.UTC().Format(time.RFC3339)
	}
	return item
}

func toPreferencesResponse(p domain.Preferences) contracts.PreferencesResponse {
	return contracts.PreferencesResponse{
		UserID:            p.UserID,
		EmailEnabled:      p.EmailEnabled,
		PushEnabled:       p.PushEnabled,
		SMSEnabled:        p.SMSEnabled,
		InAppEnabled:      p.InAppEnabled,
		QuietHoursEnabled: p.QuietHoursEnabled,
		QuietHoursStart:   p.QuietHoursStart,
		QuietHoursEnd:     p.QuietHoursEnd,
		QuietHoursTZ:      p.QuietHoursTZ,
		Language:          p.Language,
		BatchingEnabled:   p.BatchingEnabled,
		MutedTypes:        append([]string(nil), p.MutedTypes...),
		UpdatedAt:         p.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
