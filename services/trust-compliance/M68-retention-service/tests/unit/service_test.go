package unit

import (
	"context"
	"testing"

	"github.com/viralforge/mesh/services/trust-compliance/M68-retention-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/trust-compliance/M68-retention-service/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M68-retention-service/internal/domain"
)

func TestCreatePolicyAndIdempotentReplay(t *testing.T) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Policies:     repos.Policies,
		Previews:     repos.Previews,
		Holds:        repos.Holds,
		Restorations: repos.Restorations,
		Deletions:    repos.Deletions,
		Audit:        repos.Audit,
		Idempotency:  repos.Idempotency,
	})
	actor := application.Actor{SubjectID: "admin-1", Role: "admin", IdempotencyKey: "idem-policy-1"}

	row, err := svc.CreatePolicy(context.Background(), actor, application.CreatePolicyInput{
		DataType:            "user_profile",
		RetentionYears:      7,
		SoftDeleteGraceDays: 30,
	})
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	if row.Status != domain.PolicyStatusActive {
		t.Fatalf("unexpected policy: %+v", row)
	}

	replay, err := svc.CreatePolicy(context.Background(), actor, application.CreatePolicyInput{
		DataType:            "user_profile",
		RetentionYears:      7,
		SoftDeleteGraceDays: 30,
	})
	if err != nil {
		t.Fatalf("idempotent replay: %v", err)
	}
	if replay.PolicyID != row.PolicyID {
		t.Fatalf("expected replay to reuse policy id, got first=%s replay=%s", row.PolicyID, replay.PolicyID)
	}
}

func TestPreviewApprovalAndComplianceReport(t *testing.T) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Policies:     repos.Policies,
		Previews:     repos.Previews,
		Holds:        repos.Holds,
		Restorations: repos.Restorations,
		Deletions:    repos.Deletions,
		Audit:        repos.Audit,
		Idempotency:  repos.Idempotency,
	})

	policy, err := svc.CreatePolicy(context.Background(), application.Actor{SubjectID: "legal-1", Role: "legal", IdempotencyKey: "idem-policy-2"}, application.CreatePolicyInput{
		DataType:            "messages",
		RetentionYears:      5,
		SoftDeleteGraceDays: 14,
	})
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}

	preview, err := svc.CreatePreview(context.Background(), application.Actor{SubjectID: "legal-1", Role: "legal"}, application.CreatePreviewInput{
		PolicyID: policy.PolicyID,
	})
	if err != nil {
		t.Fatalf("create preview: %v", err)
	}

	deletion, err := svc.ApprovePreview(context.Background(), application.Actor{SubjectID: "legal-1", Role: "legal"}, preview.PreviewID, "routine retention")
	if err != nil {
		t.Fatalf("approve preview: %v", err)
	}
	if deletion.Status != domain.ScheduledDeletionStatusScheduled {
		t.Fatalf("unexpected deletion row: %+v", deletion)
	}

	stats, err := svc.ComplianceReport(context.Background(), application.Actor{SubjectID: "support-1", Role: "support"})
	if err != nil {
		t.Fatalf("compliance report: %v", err)
	}
	if stats["policy_count"] != 1 || stats["pending_deletions"] != 1 {
		t.Fatalf("unexpected report: %+v", stats)
	}
}

func TestLegalHoldAndRestoration(t *testing.T) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Policies:     repos.Policies,
		Previews:     repos.Previews,
		Holds:        repos.Holds,
		Restorations: repos.Restorations,
		Deletions:    repos.Deletions,
		Audit:        repos.Audit,
		Idempotency:  repos.Idempotency,
	})

	hold, err := svc.CreateLegalHold(context.Background(), application.Actor{SubjectID: "legal-2", Role: "legal", IdempotencyKey: "idem-hold-1"}, application.CreateLegalHoldInput{
		EntityID: "user-77",
		DataType: "payments",
		Reason:   "litigation",
	})
	if err != nil {
		t.Fatalf("create legal hold: %v", err)
	}
	if hold.Status != domain.LegalHoldStatusActive {
		t.Fatalf("unexpected legal hold: %+v", hold)
	}

	restoration, err := svc.CreateRestoration(context.Background(), application.Actor{SubjectID: "admin-9", Role: "admin", IdempotencyKey: "idem-restore-1"}, application.CreateRestorationInput{
		EntityID: "user-77",
		DataType: "payments",
		Reason:   "approved recovery",
	})
	if err != nil {
		t.Fatalf("create restoration: %v", err)
	}

	approved, err := svc.ApproveRestoration(context.Background(), application.Actor{SubjectID: "admin-9", Role: "admin"}, restoration.RestorationID, "approved")
	if err != nil {
		t.Fatalf("approve restoration: %v", err)
	}
	if approved.Status != domain.RestorationStatusApproved {
		t.Fatalf("unexpected restoration: %+v", approved)
	}
}
