package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M69-legal-service/internal/domain"
)

var idCounter uint64

func nextID(prefix string) string {
	n := atomic.AddUint64(&idCounter, 1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UTC().UnixNano(), n)
}

func (s *Service) UploadDocument(ctx context.Context, actor Actor, in UploadDocumentInput) (domain.LegalDocument, error) {
	if !canOperate(actor) {
		return domain.LegalDocument{}, authorizeError(actor)
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.LegalDocument{}, domain.ErrIdempotencyRequired
	}
	in.DocumentType = strings.TrimSpace(in.DocumentType)
	in.FileName = strings.TrimSpace(in.FileName)
	if in.DocumentType == "" || in.FileName == "" {
		return domain.LegalDocument{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]string{"op": "upload_document", "document_type": in.DocumentType, "file_name": in.FileName})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.LegalDocument{}, err
	} else if ok {
		var out domain.LegalDocument
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.LegalDocument{}, err
	}
	row := domain.LegalDocument{
		DocumentID:   nextID("doc"),
		DocumentType: in.DocumentType,
		FileName:     in.FileName,
		Status:       domain.DocumentStatusUploaded,
		UploadedBy:   actor.SubjectID,
		CreatedAt:    s.nowFn(),
	}
	if err := s.documents.Create(ctx, row); err != nil {
		return domain.LegalDocument{}, err
	}
	s.appendAudit(ctx, "legal.document_uploaded", actor.SubjectID, row.DocumentID, map[string]string{"document_type": row.DocumentType})
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) RequestSignature(ctx context.Context, actor Actor, documentID string, in RequestSignatureInput) (domain.SignatureRequest, error) {
	if !canOperate(actor) {
		return domain.SignatureRequest{}, authorizeError(actor)
	}
	documentID = strings.TrimSpace(documentID)
	in.SignerUserID = strings.TrimSpace(in.SignerUserID)
	if documentID == "" || in.SignerUserID == "" {
		return domain.SignatureRequest{}, domain.ErrInvalidInput
	}
	if _, err := s.documents.GetByID(ctx, documentID); err != nil {
		return domain.SignatureRequest{}, err
	}
	row := domain.SignatureRequest{
		SignatureID:  nextID("sig"),
		DocumentID:   documentID,
		SignerUserID: in.SignerUserID,
		Status:       domain.SignatureStatusRequested,
		RequestedBy:  actor.SubjectID,
		RequestedAt:  s.nowFn(),
	}
	if err := s.signatures.Create(ctx, row); err != nil {
		return domain.SignatureRequest{}, err
	}
	s.appendAudit(ctx, "legal.signature_requested", actor.SubjectID, row.SignatureID, map[string]string{"document_id": documentID})
	return row, nil
}

func (s *Service) CreateHold(ctx context.Context, actor Actor, in CreateHoldInput) (domain.LegalHold, error) {
	if !canOperate(actor) {
		return domain.LegalHold{}, authorizeError(actor)
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.LegalHold{}, domain.ErrIdempotencyRequired
	}
	in.EntityType = strings.TrimSpace(in.EntityType)
	in.EntityID = strings.TrimSpace(in.EntityID)
	in.Reason = strings.TrimSpace(in.Reason)
	if in.EntityType == "" || in.EntityID == "" || in.Reason == "" {
		return domain.LegalHold{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]string{"op": "create_hold", "entity_type": in.EntityType, "entity_id": in.EntityID, "reason": in.Reason})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.LegalHold{}, err
	} else if ok {
		var out domain.LegalHold
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.LegalHold{}, err
	}
	row := domain.LegalHold{
		HoldID:     nextID("hold"),
		EntityType: in.EntityType,
		EntityID:   in.EntityID,
		Reason:     in.Reason,
		Status:     domain.HoldStatusActive,
		IssuedBy:   actor.SubjectID,
		CreatedAt:  s.nowFn(),
	}
	if err := s.holds.Create(ctx, row); err != nil {
		return domain.LegalHold{}, err
	}
	s.appendAudit(ctx, "legal.hold_issued", actor.SubjectID, row.HoldID, map[string]string{"entity_id": row.EntityID})
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) CheckHold(ctx context.Context, actor Actor, entityType, entityID string) (bool, *domain.LegalHold, error) {
	if !canCheckHold(actor) {
		return false, nil, authorizeError(actor)
	}
	entityType = strings.TrimSpace(entityType)
	entityID = strings.TrimSpace(entityID)
	if entityType == "" || entityID == "" {
		return false, nil, domain.ErrInvalidInput
	}
	hold, err := s.holds.GetActiveByEntity(ctx, entityType, entityID)
	if err != nil {
		return false, nil, err
	}
	return hold != nil, hold, nil
}

func (s *Service) ReleaseHold(ctx context.Context, actor Actor, holdID, reason string) (domain.LegalHold, error) {
	if !canOperate(actor) {
		return domain.LegalHold{}, authorizeError(actor)
	}
	holdID = strings.TrimSpace(holdID)
	reason = strings.TrimSpace(reason)
	if holdID == "" || reason == "" {
		return domain.LegalHold{}, domain.ErrInvalidInput
	}
	row, err := s.holds.GetByID(ctx, holdID)
	if err != nil {
		return domain.LegalHold{}, err
	}
	if row.Status == domain.HoldStatusReleased {
		return row, nil
	}
	now := s.nowFn()
	row.Status = domain.HoldStatusReleased
	row.ReleasedAt = &now
	if err := s.holds.Update(ctx, row); err != nil {
		return domain.LegalHold{}, err
	}
	s.appendAudit(ctx, "legal.hold_released", actor.SubjectID, row.HoldID, map[string]string{"reason": reason})
	return row, nil
}

func (s *Service) RunComplianceScan(ctx context.Context, actor Actor, in ComplianceScanInput) (domain.ComplianceReport, error) {
	if !canOperate(actor) {
		return domain.ComplianceReport{}, authorizeError(actor)
	}
	reportType := strings.TrimSpace(in.ReportType)
	if reportType == "" {
		reportType = "daily_scan"
	}
	report := domain.ComplianceReport{
		ReportID:      nextID("scan"),
		ReportType:    reportType,
		Status:        domain.ComplianceStatusCompleted,
		FindingsCount: 2,
		DownloadURL:   "https://downloads.example.com/legal/compliance/" + reportType + ".pdf",
		CreatedBy:     actor.SubjectID,
		CreatedAt:     s.nowFn(),
	}
	if err := s.compliance.CreateReport(ctx, report); err != nil {
		return domain.ComplianceReport{}, err
	}
	findings := []domain.ComplianceFinding{
		{FindingID: nextID("finding"), ReportID: report.ReportID, Regulation: "GDPR", Severity: "high", Status: "open", Summary: "Data retention notice mismatch", CreatedAt: s.nowFn()},
		{FindingID: nextID("finding"), ReportID: report.ReportID, Regulation: "SOC2", Severity: "medium", Status: "open", Summary: "Audit evidence refresh required", CreatedAt: s.nowFn()},
	}
	for _, finding := range findings {
		_ = s.compliance.CreateFinding(ctx, finding)
	}
	s.appendAudit(ctx, "legal.compliance_scan_completed", actor.SubjectID, report.ReportID, map[string]string{"report_type": report.ReportType})
	return report, nil
}

func (s *Service) GetComplianceReport(ctx context.Context, actor Actor, reportID string) (domain.ComplianceReport, error) {
	if !canRead(actor) {
		return domain.ComplianceReport{}, authorizeError(actor)
	}
	reportID = strings.TrimSpace(reportID)
	if reportID == "" {
		return domain.ComplianceReport{}, domain.ErrInvalidInput
	}
	return s.compliance.GetReportByID(ctx, reportID)
}

func (s *Service) CreateDispute(ctx context.Context, actor Actor, in CreateDisputeInput) (domain.Dispute, error) {
	if !canDispute(actor, in.UserID) {
		return domain.Dispute{}, authorizeError(actor)
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Dispute{}, domain.ErrIdempotencyRequired
	}
	in.UserID = strings.TrimSpace(in.UserID)
	in.OpposingParty = strings.TrimSpace(in.OpposingParty)
	in.DisputeReason = strings.TrimSpace(in.DisputeReason)
	if in.UserID == "" || in.OpposingParty == "" || in.DisputeReason == "" {
		return domain.Dispute{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]any{"op": "create_dispute", "user_id": in.UserID, "opposing_party": in.OpposingParty, "dispute_reason": in.DisputeReason, "amount_cents": in.AmountCents})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Dispute{}, err
	} else if ok {
		var out domain.Dispute
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Dispute{}, err
	}
	row := domain.Dispute{
		DisputeID:     nextID("dispute"),
		UserID:        in.UserID,
		OpposingParty: in.OpposingParty,
		DisputeReason: in.DisputeReason,
		AmountCents:   in.AmountCents,
		Status:        domain.DisputeStatusOpen,
		EvidenceCount: 0,
		CreatedAt:     s.nowFn(),
	}
	if err := s.disputes.Create(ctx, row); err != nil {
		return domain.Dispute{}, err
	}
	s.appendAudit(ctx, "legal.dispute_created", actor.SubjectID, row.DisputeID, map[string]string{"user_id": row.UserID})
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) GetDispute(ctx context.Context, actor Actor, disputeID string) (domain.Dispute, error) {
	if !canRead(actor) {
		return domain.Dispute{}, authorizeError(actor)
	}
	disputeID = strings.TrimSpace(disputeID)
	if disputeID == "" {
		return domain.Dispute{}, domain.ErrInvalidInput
	}
	return s.disputes.GetByID(ctx, disputeID)
}

func (s *Service) CreateDMCANotice(ctx context.Context, actor Actor, in CreateDMCANoticeInput) (domain.DMCANotice, error) {
	if !canOperate(actor) {
		return domain.DMCANotice{}, authorizeError(actor)
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.DMCANotice{}, domain.ErrIdempotencyRequired
	}
	in.ContentID = strings.TrimSpace(in.ContentID)
	in.Claimant = strings.TrimSpace(in.Claimant)
	in.Reason = strings.TrimSpace(in.Reason)
	if in.ContentID == "" || in.Claimant == "" || in.Reason == "" {
		return domain.DMCANotice{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]string{"op": "create_dmca_notice", "content_id": in.ContentID, "claimant": in.Claimant, "reason": in.Reason})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.DMCANotice{}, err
	} else if ok {
		var out domain.DMCANotice
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.DMCANotice{}, err
	}
	row := domain.DMCANotice{
		NoticeID:   nextID("notice"),
		ContentID:  in.ContentID,
		Claimant:   in.Claimant,
		Reason:     in.Reason,
		Status:     domain.DMCANoticeStatusReceived,
		ReceivedAt: s.nowFn(),
	}
	if err := s.dmca.Create(ctx, row); err != nil {
		return domain.DMCANotice{}, err
	}
	s.appendAudit(ctx, "legal.dmca_notice_received", actor.SubjectID, row.NoticeID, map[string]string{"content_id": row.ContentID})
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) Generate1099(ctx context.Context, actor Actor, in GenerateFilingInput) (domain.RegulatoryFiling, error) {
	if !canOperate(actor) {
		return domain.RegulatoryFiling{}, authorizeError(actor)
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.RegulatoryFiling{}, domain.ErrIdempotencyRequired
	}
	in.UserID = strings.TrimSpace(in.UserID)
	if in.UserID == "" || in.TaxYear <= 0 {
		return domain.RegulatoryFiling{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]any{"op": "generate_1099", "user_id": in.UserID, "tax_year": in.TaxYear})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.RegulatoryFiling{}, err
	} else if ok {
		var out domain.RegulatoryFiling
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.RegulatoryFiling{}, err
	}
	row := domain.RegulatoryFiling{
		FilingID:      nextID("filing"),
		FilingType:    "1099",
		TaxYear:       in.TaxYear,
		UserID:        in.UserID,
		Status:        domain.FilingStatusPending,
		TaxDocumentID: nextID("taxdoc"),
		CreatedAt:     s.nowFn(),
	}
	if err := s.filings.Create(ctx, row); err != nil {
		return domain.RegulatoryFiling{}, err
	}
	s.appendAudit(ctx, "legal.regulatory_filing_created", actor.SubjectID, row.FilingID, map[string]string{"user_id": row.UserID})
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) GetFilingStatus(ctx context.Context, actor Actor, filingID string) (domain.RegulatoryFiling, error) {
	if !canRead(actor) {
		return domain.RegulatoryFiling{}, authorizeError(actor)
	}
	filingID = strings.TrimSpace(filingID)
	if filingID == "" {
		return domain.RegulatoryFiling{}, domain.ErrInvalidInput
	}
	return s.filings.GetByID(ctx, filingID)
}

func canRead(actor Actor) bool {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(actor.Role)) {
	case "admin", "legal", "support", "service":
		return true
	default:
		return false
	}
}

func canOperate(actor Actor) bool {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(actor.Role)) {
	case "admin", "legal":
		return true
	default:
		return false
	}
}

func canCheckHold(actor Actor) bool {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(actor.Role)) {
	case "admin", "legal", "support", "service":
		return true
	default:
		return false
	}
}

func canDispute(actor Actor, userID string) bool {
	userID = strings.TrimSpace(userID)
	if strings.TrimSpace(actor.SubjectID) == "" || userID == "" {
		return false
	}
	role := strings.ToLower(strings.TrimSpace(actor.Role))
	return actor.SubjectID == userID || role == "admin" || role == "legal" || role == "support"
}

func authorizeError(actor Actor) error {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.ErrUnauthorized
	}
	return domain.ErrForbidden
}

func (s *Service) appendAudit(ctx context.Context, eventType, actorID, entityID string, metadata map[string]string) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Append(ctx, domain.AuditLog{
		AuditID:    nextID("audit"),
		EventType:  eventType,
		ActorID:    actorID,
		EntityID:   entityID,
		OccurredAt: s.nowFn(),
		Metadata:   metadata,
	})
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
