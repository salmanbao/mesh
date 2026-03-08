package unit

import (
	"context"
	"testing"

	"github.com/viralforge/mesh/services/trust-compliance/M50-consent-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/trust-compliance/M50-consent-service/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M50-consent-service/internal/domain"
)

func TestUpdateConsentAndIdempotentReplay(t *testing.T) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Consents:    repos.Consents,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})

	actor := application.Actor{SubjectID: "user-1", Role: "user", IdempotencyKey: "idem-update-1"}
	row, err := svc.UpdateConsent(context.Background(), actor, application.UpdateConsentInput{
		UserID: "user-1",
		Preferences: map[string]bool{
			"marketing": true,
			"analytics": true,
		},
		Reason: "onboarding",
	})
	if err != nil {
		t.Fatalf("update consent: %v", err)
	}
	if row.Status != domain.ConsentStatusActive {
		t.Fatalf("unexpected status: %s", row.Status)
	}

	replay, err := svc.UpdateConsent(context.Background(), actor, application.UpdateConsentInput{
		UserID: "user-1",
		Preferences: map[string]bool{
			"marketing": true,
			"analytics": true,
		},
		Reason: "onboarding",
	})
	if err != nil {
		t.Fatalf("idempotent replay: %v", err)
	}
	if replay.UpdatedAt != row.UpdatedAt {
		t.Fatalf("expected replay to reuse response, got first=%s replay=%s", row.UpdatedAt, replay.UpdatedAt)
	}
}

func TestWithdrawConsentAndHistory(t *testing.T) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Consents:    repos.Consents,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})

	_, err := svc.UpdateConsent(context.Background(), application.Actor{
		SubjectID:      "admin-1",
		Role:           "admin",
		IdempotencyKey: "idem-update-admin",
	}, application.UpdateConsentInput{
		UserID: "target-1",
		Preferences: map[string]bool{
			"marketing": true,
		},
		Reason: "support override",
	})
	if err != nil {
		t.Fatalf("seed update consent: %v", err)
	}

	row, err := svc.WithdrawConsent(context.Background(), application.Actor{
		SubjectID:      "admin-1",
		Role:           "admin",
		IdempotencyKey: "idem-withdraw-1",
	}, application.WithdrawConsentInput{
		UserID:   "target-1",
		Category: "marketing",
		Reason:   "user request",
	})
	if err != nil {
		t.Fatalf("withdraw consent: %v", err)
	}
	if row.Preferences["marketing"] {
		t.Fatalf("expected marketing consent withdrawn, got %+v", row.Preferences)
	}

	history, err := svc.ListHistory(context.Background(), application.Actor{SubjectID: "target-1", Role: "user"}, "target-1", 10)
	if err != nil {
		t.Fatalf("list history: %v", err)
	}
	if len(history) < 2 {
		t.Fatalf("expected at least 2 history rows, got %d", len(history))
	}
}

func TestSupportCanViewButCannotMutateOtherUserConsent(t *testing.T) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Consents:    repos.Consents,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})

	_, _ = svc.UpdateConsent(context.Background(), application.Actor{
		SubjectID:      "admin-1",
		Role:           "admin",
		IdempotencyKey: "idem-seed",
	}, application.UpdateConsentInput{
		UserID:      "u-target",
		Preferences: map[string]bool{"analytics": true},
		Reason:      "seed",
	})

	if _, err := svc.GetConsent(context.Background(), application.Actor{SubjectID: "support-1", Role: "support"}, "u-target"); err != nil {
		t.Fatalf("support should be able to view consent: %v", err)
	}

	_, err := svc.UpdateConsent(context.Background(), application.Actor{
		SubjectID:      "support-1",
		Role:           "support",
		IdempotencyKey: "idem-forbidden",
	}, application.UpdateConsentInput{
		UserID:      "u-target",
		Preferences: map[string]bool{"analytics": false},
		Reason:      "forbidden update",
	})
	if err != domain.ErrForbidden {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}
