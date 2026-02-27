package unit

import (
	"context"
	"testing"

	"github.com/viralforge/mesh/services/trust-compliance/M38-content-license-verification/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/trust-compliance/M38-content-license-verification/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M38-content-license-verification/internal/domain"
)

func TestScanAppliesHoldForHighConfidenceMatch(t *testing.T) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Matches:     repos.Matches,
		Holds:       repos.Holds,
		Appeals:     repos.Appeals,
		Takedowns:   repos.Takedowns,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})
	actor := application.Actor{SubjectID: "creator-1", Role: "user", IdempotencyKey: "idem-scan-1"}

	out, err := svc.ScanLicense(context.Background(), actor, application.ScanLicenseInput{
		SubmissionID: "sub-1",
		CreatorID:    "creator-1",
		MediaType:    "video",
		MediaURL:     "https://cdn.example.com/copyrighted/video.mp4",
	})
	if err != nil {
		t.Fatalf("scan license: %v", err)
	}
	if out.Decision != domain.LicenseDecisionHeld || out.HoldID == "" {
		t.Fatalf("expected held decision with hold id, got: %+v", out)
	}

	replay, err := svc.ScanLicense(context.Background(), actor, application.ScanLicenseInput{
		SubmissionID: "sub-1",
		CreatorID:    "creator-1",
		MediaType:    "video",
		MediaURL:     "https://cdn.example.com/copyrighted/video.mp4",
	})
	if err != nil {
		t.Fatalf("idempotent replay: %v", err)
	}
	if replay.MatchID != out.MatchID || replay.HoldID != out.HoldID {
		t.Fatalf("idempotent replay mismatch: first=%+v replay=%+v", out, replay)
	}
}

func TestAppealAndDMCAFlow(t *testing.T) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Matches:     repos.Matches,
		Holds:       repos.Holds,
		Appeals:     repos.Appeals,
		Takedowns:   repos.Takedowns,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})

	scanActor := application.Actor{SubjectID: "creator-2", Role: "user", IdempotencyKey: "idem-scan-2"}
	scan, err := svc.ScanLicense(context.Background(), scanActor, application.ScanLicenseInput{
		SubmissionID: "sub-2",
		CreatorID:    "creator-2",
		MediaType:    "audio",
		MediaURL:     "https://cdn.example.com/copyrighted/audio.mp3",
	})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	appeal, err := svc.FileAppeal(context.Background(), application.Actor{SubjectID: "creator-2", Role: "user", IdempotencyKey: "idem-appeal-1"}, application.FileAppealInput{
		SubmissionID:       "sub-2",
		HoldID:             scan.HoldID,
		CreatorID:          "creator-2",
		CreatorExplanation: "licensed soundtrack and commentary use",
	})
	if err != nil {
		t.Fatalf("file appeal: %v", err)
	}
	if appeal.Status != domain.LicenseAppealStatusPending {
		t.Fatalf("unexpected appeal status: %s", appeal.Status)
	}

	dmca, err := svc.ReceiveDMCATakedown(context.Background(), application.Actor{SubjectID: "legal-1", Role: "legal", IdempotencyKey: "idem-dmca-1"}, application.DMCATakedownInput{
		SubmissionID:     "sub-2",
		RightsHolderName: "Example Records",
		ContactEmail:     "legal@example.com",
		Reference:        "DMCA-2026-001",
	})
	if err != nil {
		t.Fatalf("dmca: %v", err)
	}
	if dmca.Status != domain.DMCATakedownStatusReceived {
		t.Fatalf("unexpected dmca status: %s", dmca.Status)
	}
}
