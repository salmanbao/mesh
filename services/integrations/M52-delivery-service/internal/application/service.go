package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/integrations/M52-delivery-service/internal/domain"
)

func (s *Service) UpsertProductFile(ctx context.Context, actor Actor, in UpsertProductFileInput) (domain.ProductFile, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.ProductFile{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.ProductFile{}, domain.ErrIdempotencyRequired
	}
	in.ProductID = strings.TrimSpace(in.ProductID)
	in.FileID = strings.TrimSpace(in.FileID)
	in.FileName = strings.TrimSpace(in.FileName)
	in.ContentType = strings.TrimSpace(in.ContentType)
	in.Status = strings.TrimSpace(in.Status)
	if in.ProductID == "" || in.FileName == "" || in.SizeBytes <= 0 {
		return domain.ProductFile{}, domain.ErrInvalidInput
	}
	if in.ContentType == "" {
		in.ContentType = "application/octet-stream"
	}
	if in.Status == "" {
		in.Status = "validated"
	}
	if in.FileID == "" {
		in.FileID = "file_" + uuid.NewString()
	}
	requestHash := hashJSON(map[string]any{"op": "upsert_product_file", "actor": actor.SubjectID, "product_id": in.ProductID, "file_id": in.FileID, "file_name": in.FileName, "content_type": in.ContentType, "size_bytes": in.SizeBytes, "status": in.Status})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ProductFile{}, err
	} else if ok {
		var out domain.ProductFile
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ProductFile{}, err
	}
	now := s.nowFn()
	row := domain.ProductFile{FileID: in.FileID, ProductID: in.ProductID, FileName: in.FileName, ContentType: in.ContentType, SizeBytes: in.SizeBytes, Status: in.Status, CreatedAt: now, UpdatedAt: now}
	if existing, err := s.files.GetByProductID(ctx, in.ProductID); err == nil {
		row.CreatedAt = existing.CreatedAt
	}
	if err := s.files.Upsert(ctx, row); err != nil {
		return domain.ProductFile{}, err
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) GetDownloadLink(ctx context.Context, actor Actor, in GetDownloadLinkInput) (DownloadLinkResult, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return DownloadLinkResult{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return DownloadLinkResult{}, domain.ErrIdempotencyRequired
	}
	in.ProductID = strings.TrimSpace(in.ProductID)
	if in.ProductID == "" {
		return DownloadLinkResult{}, domain.ErrInvalidInput
	}
	if in.TokenTTLHours <= 0 {
		in.TokenTTLHours = int(s.cfg.DefaultTokenTTL.Hours())
	}
	if in.MaxDownloads <= 0 {
		in.MaxDownloads = s.cfg.DefaultMaxDownloads
	}
	if in.TokenTTLHours <= 0 || in.TokenTTLHours > 168 || in.MaxDownloads <= 0 || in.MaxDownloads > 100 {
		return DownloadLinkResult{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]any{"op": "get_download_link", "user_id": actor.SubjectID, "product_id": in.ProductID, "token_ttl_hours": in.TokenTTLHours, "max_downloads": in.MaxDownloads})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return DownloadLinkResult{}, err
	} else if ok {
		var out DownloadLinkResult
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return DownloadLinkResult{}, err
	}
	file, err := s.files.GetByProductID(ctx, in.ProductID)
	if err != nil {
		return DownloadLinkResult{}, err
	}
	now := s.nowFn()
	if existing, err := s.tokens.FindActiveByUserProduct(ctx, actor.SubjectID, in.ProductID, now); err == nil {
		remaining := max(0, existing.MaxDownloads-existing.DownloadCount)
		if remaining > 0 && !existing.Revoked && existing.ExpiresAt.After(now) {
			out := s.downloadLinkFromToken(existing, file)
			_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, out)
			return out, nil
		}
	}
	ttl := time.Duration(in.TokenTTLHours) * time.Hour
	tokenValue := strings.ReplaceAll(uuid.NewString()+uuid.NewString(), "-", "")
	token := domain.DownloadToken{TokenID: "dtok_" + uuid.NewString(), Token: tokenValue, ProductID: in.ProductID, UserID: actor.SubjectID, CreatedAt: now, ExpiresAt: now.Add(ttl), DownloadCount: 0, MaxDownloads: in.MaxDownloads, SingleUse: in.MaxDownloads == 1}
	if err := s.tokens.Create(ctx, token); err != nil {
		return DownloadLinkResult{}, err
	}
	out := s.downloadLinkFromToken(token, file)
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, out)
	return out, nil
}

func (s *Service) DownloadByToken(ctx context.Context, in DownloadRequest) (DownloadResult, error) {
	in.Token = strings.TrimSpace(in.Token)
	if in.Token == "" {
		return DownloadResult{}, domain.ErrInvalidInput
	}
	now := s.nowFn()
	if ip := strings.TrimSpace(in.IPAddress); ip != "" && s.downloads != nil {
		count, err := s.downloads.CountByIPSince(ctx, ip, now.Add(-1*time.Minute))
		if err != nil {
			return DownloadResult{}, err
		}
		if count >= 20 {
			return DownloadResult{}, domain.ErrRateLimited
		}
	}
	token, err := s.tokens.GetByToken(ctx, in.Token)
	if err != nil {
		return DownloadResult{}, err
	}
	if token.Revoked {
		return DownloadResult{}, domain.ErrAccessRevoked
	}
	if now.After(token.ExpiresAt) {
		return DownloadResult{}, domain.ErrTokenExpired
	}
	if token.DownloadCount >= token.MaxDownloads {
		return DownloadResult{}, domain.ErrDownloadLimitReached
	}
	file, err := s.files.GetByProductID(ctx, token.ProductID)
	if err != nil {
		return DownloadResult{}, err
	}
	token.DownloadCount++
	token.LastDownloadAt = &now
	if err := s.tokens.Update(ctx, token); err != nil {
		return DownloadResult{}, err
	}
	if s.downloads != nil {
		_ = s.downloads.Append(ctx, domain.DownloadEvent{DownloadID: "dl_" + uuid.NewString(), TokenID: token.TokenID, ProductID: token.ProductID, UserID: token.UserID, IPAddress: strings.TrimSpace(in.IPAddress), Timestamp: now, DownloadStatus: "completed", BytesTotal: file.SizeBytes, BytesDownloaded: file.SizeBytes, DurationMillis: 50})
	}
	return DownloadResult{ProductID: file.ProductID, FileID: file.FileID, FileName: file.FileName, ContentType: file.ContentType, BytesTotal: file.SizeBytes, DownloadsRemaining: max(0, token.MaxDownloads-token.DownloadCount)}, nil
}

func (s *Service) RevokeLinks(ctx context.Context, actor Actor, in RevokeLinksInput) (RevokeLinksResult, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return RevokeLinksResult{}, domain.ErrUnauthorized
	}
	if !isAdminOrSupport(actor) {
		return RevokeLinksResult{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return RevokeLinksResult{}, domain.ErrIdempotencyRequired
	}
	in.ProductID = strings.TrimSpace(in.ProductID)
	in.UserID = strings.TrimSpace(in.UserID)
	in.Reason = strings.TrimSpace(in.Reason)
	if in.ProductID == "" || in.UserID == "" || in.Reason == "" {
		return RevokeLinksResult{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]any{"op": "revoke_links", "actor": actor.SubjectID, "product_id": in.ProductID, "user_id": in.UserID, "reason": in.Reason})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return RevokeLinksResult{}, err
	} else if ok {
		var out RevokeLinksResult
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return RevokeLinksResult{}, err
	}
	rows, err := s.tokens.ListByProductUser(ctx, in.ProductID, in.UserID)
	if err != nil {
		return RevokeLinksResult{}, err
	}
	now := s.nowFn()
	revoked := 0
	for _, row := range rows {
		if row.Revoked {
			continue
		}
		row.Revoked = true
		row.RevokedAt = &now
		if err := s.tokens.Update(ctx, row); err != nil {
			return RevokeLinksResult{}, err
		}
		revoked++
		if s.revocations != nil {
			_ = s.revocations.Append(ctx, domain.DownloadRevocationAudit{RevocationID: "rev_" + uuid.NewString(), TokenID: row.TokenID, ProductID: row.ProductID, UserID: row.UserID, RevokedAt: now, Reason: in.Reason, RevokedBy: actor.SubjectID})
		}
	}
	out := RevokeLinksResult{ProductID: in.ProductID, UserID: in.UserID, RevokedCount: revoked, RevokedAt: now}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, out)
	return out, nil
}

func (s *Service) downloadLinkFromToken(token domain.DownloadToken, file domain.ProductFile) DownloadLinkResult {
	hours := int(token.ExpiresAt.Sub(s.nowFn()).Hours())
	if hours < 1 {
		hours = 1
	}
	return DownloadLinkResult{
		Token:              token.Token,
		DownloadURL:        fmt.Sprintf("%s/download/%s", strings.TrimRight(s.cfg.PublicBaseURL, "/"), token.Token),
		ExpiresAt:          token.ExpiresAt,
		ExpiresInHours:     hours,
		DownloadsRemaining: max(0, token.MaxDownloads-token.DownloadCount),
		SingleUse:          token.SingleUse,
		ProductName:        file.FileName,
		FileCount:          1,
		TotalSizeMB:        float64(file.SizeBytes) / (1024.0 * 1024.0),
	}
}

func isAdminOrSupport(actor Actor) bool {
	r := strings.ToLower(strings.TrimSpace(actor.Role))
	return r == "admin" || r == "support"
}

func hashJSON(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func (s *Service) getIdempotent(ctx context.Context, key, expectedHash string) ([]byte, bool, error) {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil, false, nil
	}
	rec, err := s.idempotency.Get(ctx, key, s.nowFn())
	if err != nil || rec == nil {
		return nil, false, err
	}
	if rec.RequestHash != expectedHash {
		return nil, false, domain.ErrIdempotencyConflict
	}
	if len(rec.ResponseBody) == 0 {
		return nil, false, nil
	}
	return rec.ResponseBody, true, nil
}
func (s *Service) reserveIdempotency(ctx context.Context, key, requestHash string) error {
	if s.idempotency == nil {
		return nil
	}
	return s.idempotency.Reserve(ctx, key, requestHash, s.nowFn().Add(s.cfg.IdempotencyTTL))
}
func (s *Service) completeIdempotencyJSON(ctx context.Context, key string, code int, v any) error {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	raw, _ := json.Marshal(v)
	return s.idempotency.Complete(ctx, key, code, raw, s.nowFn())
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
