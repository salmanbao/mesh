package unit

import (
	"context"
	"testing"

	"github.com/viralforge/mesh/services/trust-compliance/M69-legal-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/trust-compliance/M69-legal-service/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M69-legal-service/internal/domain"
)

func TestUploadDocumentAndIdempotentReplay(t *testing.T) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Documents:   repos.Documents,
		Signatures:  repos.Signatures,
		Holds:       repos.Holds,
		Compliance:  repos.Compliance,
		Disputes:    repos.Disputes,
		DMCA:        repos.DMCANotices,
		Filings:     repos.Filings,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})
	actor := application.Actor{SubjectID: "legal-1", Role: "legal", IdempotencyKey: "idem-doc-1"}

	row, err := svc.UploadDocument(context.Background(), actor, application.UploadDocumentInput{
		DocumentType: "terms_of_service",
		FileName:     "tos-v1.pdf",
	})
	if err != nil {
		t.Fatalf("upload document: %v", err)
	}
	if row.Status != domain.DocumentStatusUploaded {
		t.Fatalf("unexpected document state: %+v", row)
	}

	replay, err := svc.UploadDocument(context.Background(), actor, application.UploadDocumentInput{
		DocumentType: "terms_of_service",
		FileName:     "tos-v1.pdf",
	})
	if err != nil {
		t.Fatalf("upload replay: %v", err)
	}
	if replay.DocumentID != row.DocumentID {
		t.Fatalf("expected replay to reuse document id, got first=%s replay=%s", row.DocumentID, replay.DocumentID)
	}
}

func TestHoldLifecycleAndCheck(t *testing.T) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Documents:   repos.Documents,
		Signatures:  repos.Signatures,
		Holds:       repos.Holds,
		Compliance:  repos.Compliance,
		Disputes:    repos.Disputes,
		DMCA:        repos.DMCANotices,
		Filings:     repos.Filings,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})

	hold, err := svc.CreateHold(context.Background(), application.Actor{SubjectID: "legal-2", Role: "legal", IdempotencyKey: "idem-hold-1"}, application.CreateHoldInput{
		EntityType: "user",
		EntityID:   "user-77",
		Reason:     "litigation",
	})
	if err != nil {
		t.Fatalf("create hold: %v", err)
	}
	if hold.Status != domain.HoldStatusActive {
		t.Fatalf("unexpected hold state: %+v", hold)
	}

	held, found, err := svc.CheckHold(context.Background(), application.Actor{SubjectID: "retention", Role: "service"}, "user", "user-77")
	if err != nil {
		t.Fatalf("check hold: %v", err)
	}
	if !held || found == nil || found.HoldID != hold.HoldID {
		t.Fatalf("expected active hold, got held=%v found=%+v", held, found)
	}

	released, err := svc.ReleaseHold(context.Background(), application.Actor{SubjectID: "legal-2", Role: "legal"}, hold.HoldID, "case closed")
	if err != nil {
		t.Fatalf("release hold: %v", err)
	}
	if released.Status != domain.HoldStatusReleased {
		t.Fatalf("unexpected released hold: %+v", released)
	}
}

func TestComplianceDisputeDmcaAndFilingFlows(t *testing.T) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Documents:   repos.Documents,
		Signatures:  repos.Signatures,
		Holds:       repos.Holds,
		Compliance:  repos.Compliance,
		Disputes:    repos.Disputes,
		DMCA:        repos.DMCANotices,
		Filings:     repos.Filings,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})

	report, err := svc.RunComplianceScan(context.Background(), application.Actor{SubjectID: "legal-3", Role: "legal"}, application.ComplianceScanInput{})
	if err != nil {
		t.Fatalf("run compliance scan: %v", err)
	}
	if report.Status != domain.ComplianceStatusCompleted {
		t.Fatalf("unexpected report: %+v", report)
	}

	dispute, err := svc.CreateDispute(context.Background(), application.Actor{SubjectID: "user-9", Role: "user", IdempotencyKey: "idem-dispute-1"}, application.CreateDisputeInput{
		UserID:        "user-9",
		OpposingParty: "seller-1",
		DisputeReason: "payment_dispute",
		AmountCents:   1999,
	})
	if err != nil {
		t.Fatalf("create dispute: %v", err)
	}
	if dispute.Status != domain.DisputeStatusOpen {
		t.Fatalf("unexpected dispute: %+v", dispute)
	}

	notice, err := svc.CreateDMCANotice(context.Background(), application.Actor{SubjectID: "legal-3", Role: "legal", IdempotencyKey: "idem-dmca-1"}, application.CreateDMCANoticeInput{
		ContentID: "content-1",
		Claimant:  "Studio",
		Reason:    "copyright infringement",
	})
	if err != nil {
		t.Fatalf("create dmca notice: %v", err)
	}
	if notice.Status != domain.DMCANoticeStatusReceived {
		t.Fatalf("unexpected dmca notice: %+v", notice)
	}

	filing, err := svc.Generate1099(context.Background(), application.Actor{SubjectID: "legal-3", Role: "legal", IdempotencyKey: "idem-filing-1"}, application.GenerateFilingInput{
		UserID:  "user-9",
		TaxYear: 2025,
	})
	if err != nil {
		t.Fatalf("generate filing: %v", err)
	}
	if filing.FilingType != "1099" {
		t.Fatalf("unexpected filing: %+v", filing)
	}
}
