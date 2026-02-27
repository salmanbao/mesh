package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/integrations/M11-distribution-tracking-service/internal/application"
	"github.com/viralforge/mesh/services/integrations/M11-distribution-tracking-service/internal/contracts"
	"github.com/viralforge/mesh/services/integrations/M11-distribution-tracking-service/internal/domain"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) validatePost(w http.ResponseWriter, r *http.Request) {
	var req contracts.ValidatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	out, err := h.service.ValidatePost(r.Context(), actorFromContext(r.Context()), application.ValidatePostInput{UserID: req.UserID, Platform: req.Platform, PostURL: req.PostURL})
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "post validated", out)
}

func (h *Handler) registerPost(w http.ResponseWriter, r *http.Request) {
	var req contracts.RegisterPostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	post, pending, err := h.service.RegisterPost(r.Context(), actorFromContext(r.Context()), application.RegisterPostInput{UserID: req.UserID, Platform: req.Platform, PostURL: req.PostURL, DistributionItemID: req.DistributionItemID, CampaignID: req.CampaignID})
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	statusCode := http.StatusOK
	msg := "tracked post registered"
	if pending {
		statusCode = http.StatusAccepted
		msg = "tracked post accepted (attribution pending)"
	}
	writeSuccess(w, statusCode, msg, toTrackedPostResponse(post))
}

func (h *Handler) getPost(w http.ResponseWriter, r *http.Request) {
	post, err := h.service.GetTrackedPost(r.Context(), actorFromContext(r.Context()), chi.URLParam(r, "id"))
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "tracked post", toTrackedPostResponse(post))
}

func (h *Handler) getMetrics(w http.ResponseWriter, r *http.Request) {
	post, snaps, err := h.service.GetTrackedPostMetrics(r.Context(), actorFromContext(r.Context()), chi.URLParam(r, "id"))
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	resp := contracts.MetricsListResponse{TrackedPostID: post.TrackedPostID, Snapshots: make([]contracts.MetricSnapshotResponse, 0, len(snaps))}
	if post.LastPolledAt != nil {
		resp.LastPolledAt = post.LastPolledAt.UTC().Format(time.RFC3339)
	}
	for _, s := range snaps {
		resp.Snapshots = append(resp.Snapshots, contracts.MetricSnapshotResponse{SnapshotID: s.SnapshotID, TrackedPostID: s.TrackedPostID, Platform: s.Platform, Views: s.Views, Likes: s.Likes, Shares: s.Shares, Comments: s.Comments, PolledAt: s.PolledAt.UTC().Format(time.RFC3339)})
	}
	writeSuccess(w, http.StatusOK, "tracked post metrics", resp)
}

func toTrackedPostResponse(p domain.TrackedPost) contracts.TrackedPostResponse {
	out := contracts.TrackedPostResponse{TrackedPostID: p.TrackedPostID, UserID: p.UserID, Platform: p.Platform, PostURL: p.PostURL, DistributionItemID: p.DistributionItemID, CampaignID: p.CampaignID, Status: string(p.Status), ValidationStatus: p.ValidationStatus, CreatedAt: p.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: p.UpdatedAt.UTC().Format(time.RFC3339)}
	if p.LastPolledAt != nil {
		out.LastPolledAt = p.LastPolledAt.UTC().Format(time.RFC3339)
	}
	return out
}
