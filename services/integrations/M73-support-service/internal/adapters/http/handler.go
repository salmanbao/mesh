package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/viralforge/mesh/services/integrations/M73-support-service/internal/application"
	"github.com/viralforge/mesh/services/integrations/M73-support-service/internal/contracts"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) createTicket(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	ticket, err := h.service.CreateTicket(r.Context(), actor, application.CreateTicketInput{
		Subject:     req.Subject,
		Description: req.Description,
		Category:    req.Category,
		Priority:    req.Priority,
		Channel:     req.Channel,
		EntityType:  req.EntityType,
		EntityID:    req.EntityID,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", ticket)
}

func (h *Handler) createTicketFromEmail(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.CreateFromEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	ticket, err := h.service.CreateTicketFromEmail(r.Context(), actor, application.CreateFromEmailInput(req))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", ticket)
}

func (h *Handler) getTicket(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	ticket, err := h.service.GetTicket(r.Context(), actor, r.PathValue("id"))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", ticket)
}

func (h *Handler) searchTickets(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			limit = v
		}
	}
	items, err := h.service.SearchTickets(r.Context(), actor, application.SearchTicketsInput{
		Query:      strings.TrimSpace(r.URL.Query().Get("q")),
		Status:     strings.TrimSpace(r.URL.Query().Get("status")),
		Category:   strings.TrimSpace(r.URL.Query().Get("category")),
		UserID:     strings.TrimSpace(r.URL.Query().Get("user_id")),
		AssignedTo: strings.TrimSpace(r.URL.Query().Get("assigned_to")),
		Limit:      limit,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", items)
}

func (h *Handler) updateTicket(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.UpdateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	ticket, err := h.service.UpdateTicket(r.Context(), actor, r.PathValue("id"), application.UpdateTicketInput(req))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", ticket)
}

func (h *Handler) deleteTicket(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	ticket, err := h.service.DeleteTicket(r.Context(), actor, r.PathValue("id"))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", ticket)
}

func (h *Handler) assignTicket(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.AssignTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	ticket, err := h.service.AssignTicket(r.Context(), actor, r.PathValue("id"), application.AssignTicketInput(req))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", ticket)
}

func (h *Handler) addReply(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.AddReplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	reply, err := h.service.AddReply(r.Context(), actor, r.PathValue("id"), application.AddReplyInput(req))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", reply)
}

func (h *Handler) submitCSAT(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.SubmitCSATRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	rating, err := h.service.SubmitCSAT(r.Context(), actor, r.PathValue("id"), application.SubmitCSATInput(req))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", rating)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeSuccess(w, http.StatusOK, "ok", nil)
}
