package http

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/integrations/M52-delivery-service/internal/application"
	"github.com/viralforge/mesh/services/integrations/M52-delivery-service/internal/contracts"
	"github.com/viralforge/mesh/services/integrations/M52-delivery-service/internal/domain"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) upsertProductFile(w http.ResponseWriter, r *http.Request) {
	var req contracts.UpsertProductFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid json body")
		return
	}
	row, err := h.service.UpsertProductFile(r.Context(), actorFromContext(r.Context()), application.UpsertProductFileInput{
		ProductID:   chi.URLParam(r, "product_id"),
		FileID:      req.FileID,
		FileName:    req.FileName,
		ContentType: req.ContentType,
		SizeBytes:   req.SizeBytes,
		Status:      req.Status,
	})
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error())
		return
	}
	writeSuccess(w, http.StatusOK, toProductFileResponse(row))
}

func (h *Handler) getDownloadLink(w http.ResponseWriter, r *http.Request) {
	in := application.GetDownloadLinkInput{ProductID: chi.URLParam(r, "product_id")}
	if raw := strings.TrimSpace(r.URL.Query().Get("token_ttl_hours")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			in.TokenTTLHours = v
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("max_downloads")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			in.MaxDownloads = v
		}
	}
	out, err := h.service.GetDownloadLink(r.Context(), actorFromContext(r.Context()), in)
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error())
		return
	}
	writeSuccess(w, http.StatusOK, contracts.DownloadLinkResponse{
		DownloadURL:        out.DownloadURL,
		ExpiresAt:          out.ExpiresAt.UTC().Format(time.RFC3339),
		ExpiresInHours:     out.ExpiresInHours,
		DownloadsRemaining: out.DownloadsRemaining,
		SingleUse:          out.SingleUse,
		ProductName:        out.ProductName,
		FileCount:          out.FileCount,
		TotalSizeMB:        out.TotalSizeMB,
		Token:              out.Token,
	})
}

func (h *Handler) downloadByToken(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	out, err := h.service.DownloadByToken(r.Context(), application.DownloadRequest{Token: chi.URLParam(r, "token"), IPAddress: ip, RangeHeader: r.Header.Get("Range")})
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error())
		return
	}
	w.Header().Set("X-Mock-Delivery", "metadata")
	writeSuccess(w, http.StatusOK, contracts.DownloadMetadataResponse{ProductID: out.ProductID, FileID: out.FileID, FileName: out.FileName, ContentType: out.ContentType, BytesTotal: out.BytesTotal, DownloadsRemaining: out.DownloadsRemaining})
}

func (h *Handler) revokeLinks(w http.ResponseWriter, r *http.Request) {
	var req contracts.RevokeLinksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid json body")
		return
	}
	out, err := h.service.RevokeLinks(r.Context(), actorFromContext(r.Context()), application.RevokeLinksInput{ProductID: req.ProductID, UserID: req.UserID, Reason: req.Reason})
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error())
		return
	}
	writeSuccess(w, http.StatusOK, contracts.RevokeLinksResponse{ProductID: out.ProductID, UserID: out.UserID, RevokedCount: out.RevokedCount, RevocationTime: out.RevokedAt.UTC().Format(time.RFC3339)})
}

func toProductFileResponse(row domain.ProductFile) contracts.ProductFileResponse {
	return contracts.ProductFileResponse{FileID: row.FileID, ProductID: row.ProductID, FileName: row.FileName, ContentType: row.ContentType, SizeBytes: row.SizeBytes, Status: row.Status, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339)}
}

func clientIP(r *http.Request) string {
	if raw := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); raw != "" {
		parts := strings.Split(raw, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}
