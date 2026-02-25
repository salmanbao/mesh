package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/application"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/ports"
)

func (h *Handler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := bearerTokenFromHeader(r.Header.Get("Authorization"))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials")
			return
		}
		claims, err := h.service.ValidateToken(r.Context(), raw)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials")
			return
		}
		ctx := r.Context()
		ctx = contextWithClaims(ctx, claims, raw)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func contextWithClaims(ctx context.Context, claims ports.AuthClaims, token string) context.Context {
	ctx = context.WithValue(ctx, ctxKeyClaims, claims)
	ctx = context.WithValue(ctx, ctxKeyTokenRaw, token)
	return ctx
}

func (h *Handler) getMyProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials")
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials")
		return
	}
	resp, err := h.service.GetMyProfile(r.Context(), userID)
	if err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeSuccess(w, http.StatusOK, resp)
}

func (h *Handler) getPublicProfile(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	var requester *uuid.UUID
	if claims, ok := claimsFromContext(r.Context()); ok && claims.UserID != "" {
		if parsed, err := uuid.Parse(claims.UserID); err == nil {
			requester = &parsed
		}
	}
	resp, err := h.service.GetPublicProfile(r.Context(), username, requester)
	if err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	if resp.RedirectTo != "" {
		http.Redirect(w, r, "/v1/profiles/"+resp.RedirectTo, http.StatusMovedPermanently)
		return
	}
	writeSuccess(w, http.StatusOK, resp.Profile)
}

func (h *Handler) updateMyProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials")
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials")
		return
	}
	var req application.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json body")
		return
	}
	resp, err := h.service.UpdateProfile(r.Context(), userID, req, r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeSuccess(w, http.StatusOK, resp)
}

func (h *Handler) changeUsername(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials")
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials")
		return
	}
	var req application.ChangeUsernameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json body")
		return
	}
	resp, err := h.service.ChangeUsername(r.Context(), userID, req, r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeSuccess(w, http.StatusOK, resp)
}

func (h *Handler) checkUsernameAvailability(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "username query param is required")
		return
	}
	resp, err := h.service.CheckUsernameAvailability(r.Context(), username)
	if err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeSuccess(w, http.StatusOK, resp)
}

func (h *Handler) uploadAvatar(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials")
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials")
		return
	}
	if err := r.ParseMultipartForm(8 * 1024 * 1024); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid multipart payload")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "file is required")
		return
	}
	defer file.Close()
	contentType := header.Header.Get("Content-Type")
	data := make([]byte, header.Size)
	_, _ = file.Read(data)

	resp, err := h.service.UploadAvatar(r.Context(), userID, header.Filename, contentType, data, r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeSuccess(w, http.StatusAccepted, resp)
}

func (h *Handler) addSocialLink(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials")
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials")
		return
	}
	var req application.AddSocialLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json body")
		return
	}
	resp, err := h.service.AddSocialLink(r.Context(), userID, req, r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeSuccess(w, http.StatusCreated, resp)
}

func (h *Handler) deleteSocialLink(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials")
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials")
		return
	}
	platform := chi.URLParam(r, "platform")
	if platform == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "platform is required")
		return
	}
	if err := h.service.DeleteSocialLink(r.Context(), userID, platform); err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeMessage(w, http.StatusOK, "Social link removed")
}

func (h *Handler) putPayoutMethod(w http.ResponseWriter, r *http.Request) {
	h.writePayoutMethod(w, r, false)
}

func (h *Handler) updatePayoutMethod(w http.ResponseWriter, r *http.Request) {
	h.writePayoutMethod(w, r, true)
}

func (h *Handler) writePayoutMethod(w http.ResponseWriter, r *http.Request, overrideFromPath bool) {
	claims, ok := claimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials")
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials")
		return
	}
	var req application.PutPayoutMethodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json body")
		return
	}
	if overrideFromPath {
		req.MethodType = chi.URLParam(r, "method_type")
	}
	resp, err := h.service.PutPayoutMethod(r.Context(), userID, req, r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	if overrideFromPath {
		writeSuccess(w, http.StatusOK, resp)
		return
	}
	writeSuccess(w, http.StatusCreated, resp)
}

func (h *Handler) uploadKYCDocument(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials")
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials")
		return
	}
	if err := r.ParseMultipartForm(16 * 1024 * 1024); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid multipart payload")
		return
	}
	documentType := strings.TrimSpace(r.FormValue("document_type"))
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "file is required")
		return
	}
	defer file.Close()
	data := make([]byte, header.Size)
	_, _ = file.Read(data)

	resp, err := h.service.UploadKYCDocument(r.Context(), userID, application.UploadKYCDocumentRequest{
		DocumentType:    documentType,
		FileName:        header.Filename,
		FileContentType: header.Header.Get("Content-Type"),
		FileBytes:       data,
	}, r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeSuccess(w, http.StatusCreated, resp)
}

func (h *Handler) getKYCStatus(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials")
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing credentials")
		return
	}
	resp, err := h.service.GetKYCStatus(r.Context(), userID)
	if err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeSuccess(w, http.StatusOK, resp)
}

func (h *Handler) adminListProfiles(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromContext(r.Context())
	if !ok || strings.ToUpper(claims.Role) != "ADMIN" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "admin role required")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	resp, err := h.service.AdminListProfiles(r.Context(), limit, offset)
	if err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{
		"profiles": resp,
		"pagination": map[string]any{
			"limit":  limit,
			"offset": offset,
		},
	})
}

func (h *Handler) adminApproveKYC(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromContext(r.Context())
	if !ok || strings.ToUpper(claims.Role) != "ADMIN" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "admin role required")
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "user_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid user_id")
		return
	}
	adminID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
		return
	}
	if err := h.service.AdminApproveKYC(r.Context(), application.AdminKYCDecisionRequest{
		UserID: userID, ReviewedBy: adminID, Now: time.Now().UTC(),
	}); err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{
		"user_id":    userID.String(),
		"kyc_status": "verified",
	})
}

func (h *Handler) adminRejectKYC(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromContext(r.Context())
	if !ok || strings.ToUpper(claims.Role) != "ADMIN" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "admin role required")
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "user_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid user_id")
		return
	}
	adminID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
		return
	}
	var body struct {
		RejectionReason string `json:"rejection_reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json body")
		return
	}
	if err := h.service.AdminRejectKYC(r.Context(), application.AdminKYCDecisionRequest{
		UserID: userID, ReviewedBy: adminID, RejectionReason: body.RejectionReason, Now: time.Now().UTC(),
	}); err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{
		"user_id":          userID.String(),
		"kyc_status":       "rejected",
		"rejection_reason": body.RejectionReason,
	})
}
