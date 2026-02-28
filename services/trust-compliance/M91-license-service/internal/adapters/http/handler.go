package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/viralforge/mesh/services/trust-compliance/M91-license-service/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M91-license-service/internal/contracts"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) listLicenses(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	items, err := h.service.ListLicenses(r.Context(), actor)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", items)
}

func (h *Handler) validate(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	key := strings.TrimSpace(r.URL.Query().Get("license_key"))
	if key == "" {
		key = strings.TrimSpace(r.URL.Query().Get("key"))
	}
	out, err := h.service.Validate(r.Context(), actor, key)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", out)
}

func (h *Handler) activate(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.ActivateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	out, err := h.service.Activate(r.Context(), actor, application.ActivateInput{LicenseKey: req.LicenseKey, DeviceID: req.DeviceID, DeviceFingerprint: req.DeviceFingerprint})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", out)
}

func (h *Handler) deactivate(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.DeactivateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	out, err := h.service.Deactivate(r.Context(), actor, application.DeactivateInput{LicenseKey: req.LicenseKey, DeviceID: req.DeviceID})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", out)
}

func (h *Handler) exportLicenses(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.ExportRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	out, err := h.service.Export(r.Context(), actor, application.ExportInput{Format: req.Format})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", out)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeSuccess(w, http.StatusOK, "ok", map[string]string{"status": "ok"})
}
