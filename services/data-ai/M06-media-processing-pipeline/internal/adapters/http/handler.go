package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/application"
	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/contracts"
)

func (h *Handler) createUpload(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.UploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	out, err := h.service.CreateUpload(r.Context(), actor, application.CreateUploadInput{
		SubmissionID:   strings.TrimSpace(req.SubmissionID),
		FileName:       strings.TrimSpace(req.FileName),
		MIMEType:       strings.TrimSpace(req.MIMEType),
		FileSize:       req.FileSize,
		ChecksumSHA256: strings.TrimSpace(req.ChecksumSHA256),
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", contracts.UploadResponse{AssetID: out.AssetID, UploadURL: out.UploadURL, ExpiresIn: out.ExpiresIn})
}

func (h *Handler) getAsset(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	assetID := chi.URLParam(r, "asset_id")
	out, err := h.service.GetAssetStatus(r.Context(), actor, assetID)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", out)
}

func (h *Handler) retryAsset(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	assetID := chi.URLParam(r, "asset_id")
	out, err := h.service.RetryAsset(r.Context(), actor, application.RetryAssetInput{AssetID: assetID})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", out)
}
