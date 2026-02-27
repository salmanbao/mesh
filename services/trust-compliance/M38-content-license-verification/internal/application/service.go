package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"
	"sync/atomic"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M38-content-license-verification/internal/contracts"
	"github.com/viralforge/mesh/services/trust-compliance/M38-content-license-verification/internal/domain"
)

var idCounter uint64

func nextID(prefix string) string {
	n := atomic.AddUint64(&idCounter, 1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UTC().UnixNano(), n)
}

func (s *Service) ScanLicense(ctx context.Context, actor Actor, in ScanLicenseInput) (contracts.ScanLicenseResponse, error) {
	creatorID, err := s.resolveCreator(actor, in.CreatorID)
	if err != nil {
		return contracts.ScanLicenseResponse{}, err
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return contracts.ScanLicenseResponse{}, domain.ErrIdempotencyRequired
	}
	in.SubmissionID = strings.TrimSpace(in.SubmissionID)
	in.MediaType = strings.ToLower(strings.TrimSpace(in.MediaType))
	in.MediaURL = strings.TrimSpace(in.MediaURL)
	in.DeclaredLicenseID = strings.TrimSpace(in.DeclaredLicenseID)
	if in.SubmissionID == "" || in.MediaURL == "" || !domain.IsValidMediaType(in.MediaType) {
		return contracts.ScanLicenseResponse{}, domain.ErrInvalidInput
	}

	requestHash := hashJSON(map[string]string{
		"op":                  "license_scan",
		"submission_id":       in.SubmissionID,
		"creator_id":          creatorID,
		"media_type":          in.MediaType,
		"media_url":           in.MediaURL,
		"declared_license_id": in.DeclaredLicenseID,
	})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return contracts.ScanLicenseResponse{}, err
	} else if ok {
		var out contracts.ScanLicenseResponse
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return contracts.ScanLicenseResponse{}, err
	}

	if existing, err := s.matches.GetBySubmissionID(ctx, in.SubmissionID); err == nil {
		out := contracts.ScanLicenseResponse{
			MatchID:         existing.MatchID,
			SubmissionID:    existing.SubmissionID,
			ConfidenceScore: existing.ConfidenceScore,
			Decision:        domain.LicenseDecisionAllowed,
			ScannedAt:       existing.CreatedAt.UTC().Format(time.RFC3339),
		}
		if hold, holdErr := s.holds.GetBySubmissionID(ctx, in.SubmissionID); holdErr == nil {
			out.Decision = domain.LicenseDecisionHeld
			out.HoldID = hold.HoldID
		}
		_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, out)
		return out, nil
	}

	now := s.nowFn()
	confidence := estimatedConfidence(in.MediaURL, in.DeclaredLicenseID)
	match := domain.CopyrightMatch{
		MatchID:          nextID("match"),
		SubmissionID:     in.SubmissionID,
		CreatorID:        creatorID,
		MediaType:        in.MediaType,
		MediaURL:         in.MediaURL,
		ConfidenceScore:  confidence,
		MatchedTitle:     "potential_copyright_match",
		RightsHolderName: "rights_holder_unknown",
		CreatedAt:        now,
	}
	if err := s.matches.Create(ctx, match); err != nil {
		return contracts.ScanLicenseResponse{}, err
	}

	out := contracts.ScanLicenseResponse{
		MatchID:         match.MatchID,
		SubmissionID:    match.SubmissionID,
		ConfidenceScore: match.ConfidenceScore,
		Decision:        domain.LicenseDecisionAllowed,
		ScannedAt:       now.UTC().Format(time.RFC3339),
	}
	if confidence >= s.cfg.HoldThreshold && in.DeclaredLicenseID == "" {
		hold := domain.LicenseHold{
			HoldID:        nextID("hold"),
			SubmissionID:  in.SubmissionID,
			MatchID:       match.MatchID,
			CreatorID:     creatorID,
			Reason:        "high_confidence_copyright_match",
			Status:        domain.LicenseHoldStatusPendingReview,
			HoldCreatedAt: now,
		}
		if err := s.holds.Create(ctx, hold); err != nil {
			return contracts.ScanLicenseResponse{}, err
		}
		out.Decision = domain.LicenseDecisionHeld
		out.HoldID = hold.HoldID
	}
	if s.audit != nil {
		_ = s.audit.Append(ctx, domain.AuditLog{
			EventID:      nextID("audit"),
			EventType:    "license.scan_completed",
			EntityID:     match.MatchID,
			SubmissionID: match.SubmissionID,
			ActorID:      actor.SubjectID,
			Metadata: map[string]string{
				"decision": fmt.Sprintf("%s", out.Decision),
			},
			CreatedAt: now,
		})
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, out)
	return out, nil
}

func (s *Service) FileAppeal(ctx context.Context, actor Actor, in FileAppealInput) (contracts.FileAppealResponse, error) {
	creatorID, err := s.resolveCreator(actor, in.CreatorID)
	if err != nil {
		return contracts.FileAppealResponse{}, err
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return contracts.FileAppealResponse{}, domain.ErrIdempotencyRequired
	}
	in.SubmissionID = strings.TrimSpace(in.SubmissionID)
	in.HoldID = strings.TrimSpace(in.HoldID)
	in.CreatorExplanation = strings.TrimSpace(in.CreatorExplanation)
	if in.SubmissionID == "" || in.CreatorExplanation == "" {
		return contracts.FileAppealResponse{}, domain.ErrInvalidInput
	}

	requestHash := hashJSON(map[string]string{
		"op":                  "license_appeal",
		"submission_id":       in.SubmissionID,
		"hold_id":             in.HoldID,
		"creator_id":          creatorID,
		"creator_explanation": in.CreatorExplanation,
	})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return contracts.FileAppealResponse{}, err
	} else if ok {
		var out contracts.FileAppealResponse
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return contracts.FileAppealResponse{}, err
	}

	var hold domain.LicenseHold
	if in.HoldID != "" {
		hold, err = s.holds.GetByID(ctx, in.HoldID)
	} else {
		hold, err = s.holds.GetBySubmissionID(ctx, in.SubmissionID)
	}
	if err != nil {
		return contracts.FileAppealResponse{}, err
	}
	if hold.SubmissionID != in.SubmissionID || !canActForCreator(actor, hold.CreatorID) {
		return contracts.FileAppealResponse{}, domain.ErrForbidden
	}

	now := s.nowFn()
	appeal := domain.LicenseAppeal{
		AppealID:           nextID("appeal"),
		SubmissionID:       in.SubmissionID,
		HoldID:             hold.HoldID,
		CreatorID:          creatorID,
		CreatorExplanation: in.CreatorExplanation,
		Status:             domain.LicenseAppealStatusPending,
		AppealCreatedAt:    now,
	}
	if err := s.appeals.Create(ctx, appeal); err != nil {
		return contracts.FileAppealResponse{}, err
	}
	if s.audit != nil {
		_ = s.audit.Append(ctx, domain.AuditLog{
			EventID:      nextID("audit"),
			EventType:    "license.appeal_filed",
			EntityID:     appeal.AppealID,
			SubmissionID: appeal.SubmissionID,
			ActorID:      actor.SubjectID,
			CreatedAt:    now,
		})
	}
	out := contracts.FileAppealResponse{
		AppealID:        appeal.AppealID,
		SubmissionID:    appeal.SubmissionID,
		HoldID:          appeal.HoldID,
		Status:          appeal.Status,
		AppealCreatedAt: appeal.AppealCreatedAt.UTC().Format(time.RFC3339),
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, out)
	return out, nil
}

func (s *Service) ReceiveDMCATakedown(ctx context.Context, actor Actor, in DMCATakedownInput) (contracts.DMCATakedownResponse, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return contracts.DMCATakedownResponse{}, domain.ErrUnauthorized
	}
	role := strings.ToLower(strings.TrimSpace(actor.Role))
	if role != "admin" && role != "legal" {
		return contracts.DMCATakedownResponse{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return contracts.DMCATakedownResponse{}, domain.ErrIdempotencyRequired
	}
	in.SubmissionID = strings.TrimSpace(in.SubmissionID)
	in.RightsHolderName = strings.TrimSpace(in.RightsHolderName)
	in.ContactEmail = strings.TrimSpace(in.ContactEmail)
	in.Reference = strings.TrimSpace(in.Reference)
	if in.SubmissionID == "" || in.RightsHolderName == "" || in.ContactEmail == "" || in.Reference == "" {
		return contracts.DMCATakedownResponse{}, domain.ErrInvalidInput
	}
	if _, err := mail.ParseAddress(in.ContactEmail); err != nil {
		return contracts.DMCATakedownResponse{}, domain.ErrInvalidInput
	}

	requestHash := hashJSON(map[string]string{
		"op":                 "dmca_takedown",
		"submission_id":      in.SubmissionID,
		"rights_holder_name": in.RightsHolderName,
		"contact_email":      in.ContactEmail,
		"reference":          in.Reference,
	})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return contracts.DMCATakedownResponse{}, err
	} else if ok {
		var out contracts.DMCATakedownResponse
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return contracts.DMCATakedownResponse{}, err
	}

	now := s.nowFn()
	row := domain.DMCATakedown{
		DMCAID:           nextID("dmca"),
		SubmissionID:     in.SubmissionID,
		RightsHolder:     in.RightsHolderName,
		ContactEmail:     in.ContactEmail,
		Reference:        in.Reference,
		Status:           domain.DMCATakedownStatusReceived,
		NoticeReceivedAt: now,
	}
	if err := s.takedowns.Create(ctx, row); err != nil {
		return contracts.DMCATakedownResponse{}, err
	}
	if s.audit != nil {
		_ = s.audit.Append(ctx, domain.AuditLog{
			EventID:      nextID("audit"),
			EventType:    "license.dmca_received",
			EntityID:     row.DMCAID,
			SubmissionID: row.SubmissionID,
			ActorID:      actor.SubjectID,
			CreatedAt:    now,
		})
	}
	out := contracts.DMCATakedownResponse{
		DMCAID:           row.DMCAID,
		SubmissionID:     row.SubmissionID,
		Status:           row.Status,
		NoticeReceivedAt: row.NoticeReceivedAt.UTC().Format(time.RFC3339),
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, out)
	return out, nil
}

func estimatedConfidence(mediaURL, declaredLicenseID string) float64 {
	mediaURL = strings.ToLower(strings.TrimSpace(mediaURL))
	if declaredLicenseID != "" {
		return 0.10
	}
	if strings.Contains(mediaURL, "copyrighted") || strings.Contains(mediaURL, "dmca") {
		return 0.99
	}
	if strings.Contains(mediaURL, "match") {
		return 0.95
	}
	return 0.40
}

func canActForCreator(actor Actor, creatorID string) bool {
	role := strings.ToLower(strings.TrimSpace(actor.Role))
	actorID := strings.TrimSpace(actor.SubjectID)
	creatorID = strings.TrimSpace(creatorID)
	return actorID != "" && creatorID != "" && (actorID == creatorID || role == "admin" || role == "support" || role == "legal")
}

func (s *Service) resolveCreator(actor Actor, requested string) (string, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return "", domain.ErrUnauthorized
	}
	requested = strings.TrimSpace(requested)
	if requested == "" {
		requested = strings.TrimSpace(actor.SubjectID)
	}
	if !canActForCreator(actor, requested) {
		return "", domain.ErrForbidden
	}
	return requested, nil
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
