package unit

import (
	"context"
	"testing"

	"github.com/viralforge/mesh/services/trust-compliance/M51-data-portability-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/trust-compliance/M51-data-portability-service/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M51-data-portability-service/internal/domain"
)

func TestCreateExportAndIdempotentReplay(t *testing.T) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Exports:     repos.ExportRequests,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})
	actor := application.Actor{SubjectID: "user-1", Role: "user", IdempotencyKey: "idem-export-1"}

	row, err := svc.CreateExport(context.Background(), actor, application.CreateExportInput{
		UserID: "user-1",
		Format: "json",
	})
	if err != nil {
		t.Fatalf("create export: %v", err)
	}
	if row.Status != domain.ExportStatusCompleted || row.RequestType != domain.ExportRequestTypeExport {
		t.Fatalf("unexpected export state: %+v", row)
	}

	replay, err := svc.CreateExport(context.Background(), actor, application.CreateExportInput{
		UserID: "user-1",
		Format: "json",
	})
	if err != nil {
		t.Fatalf("idempotent replay: %v", err)
	}
	if replay.RequestID != row.RequestID {
		t.Fatalf("expected same request id on replay, got first=%s replay=%s", row.RequestID, replay.RequestID)
	}
}

func TestEraseAndReadFlows(t *testing.T) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Exports:     repos.ExportRequests,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})

	erase, err := svc.CreateEraseRequest(context.Background(), application.Actor{SubjectID: "user-2", Role: "user", IdempotencyKey: "idem-erase-1"}, application.EraseInput{
		UserID: "user-2",
		Reason: "gdpr_erasure",
	})
	if err != nil {
		t.Fatalf("create erase request: %v", err)
	}
	if erase.RequestType != domain.ExportRequestTypeErase {
		t.Fatalf("unexpected erase request type: %s", erase.RequestType)
	}

	got, err := svc.GetExport(context.Background(), application.Actor{SubjectID: "user-2", Role: "user"}, erase.RequestID)
	if err != nil {
		t.Fatalf("get export: %v", err)
	}
	if got.RequestID != erase.RequestID {
		t.Fatalf("expected request id %s, got %s", erase.RequestID, got.RequestID)
	}

	rows, err := svc.ListExports(context.Background(), application.Actor{SubjectID: "user-2", Role: "user"}, "", 10)
	if err != nil {
		t.Fatalf("list exports: %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("expected at least one export row")
	}
}
