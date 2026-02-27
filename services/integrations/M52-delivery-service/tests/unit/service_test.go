package unit

import (
	"context"
	"testing"

	"github.com/viralforge/mesh/services/integrations/M52-delivery-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/integrations/M52-delivery-service/internal/application"
)

func newService() *application.Service {
	repos := postgres.NewRepositories()
	return application.NewService(application.Dependencies{
		Files: repos.Files, Tokens: repos.Tokens, Downloads: repos.Downloads, Revocations: repos.Revocations, Idempotency: repos.Idempotency, EventDedup: repos.EventDedup,
	})
}

func TestGetDownloadLinkIdempotentAndDownload(t *testing.T) {
	svc := newService()
	admin := application.Actor{SubjectID: "admin-1", Role: "admin", IdempotencyKey: "idem-file-1"}
	if _, err := svc.UpsertProductFile(context.Background(), admin, application.UpsertProductFileInput{ProductID: "prod-1", FileName: "asset.zip", ContentType: "application/zip", SizeBytes: 1024}); err != nil {
		t.Fatalf("upsert file: %v", err)
	}
	actor := application.Actor{SubjectID: "user-1", Role: "creator", IdempotencyKey: "idem-link-1"}
	first, err := svc.GetDownloadLink(context.Background(), actor, application.GetDownloadLinkInput{ProductID: "prod-1", TokenTTLHours: 24, MaxDownloads: 2})
	if err != nil {
		t.Fatalf("first link: %v", err)
	}
	second, err := svc.GetDownloadLink(context.Background(), actor, application.GetDownloadLinkInput{ProductID: "prod-1", TokenTTLHours: 24, MaxDownloads: 2})
	if err != nil {
		t.Fatalf("second link: %v", err)
	}
	if first.Token != second.Token {
		t.Fatalf("expected idempotent token reuse")
	}
	res, err := svc.DownloadByToken(context.Background(), application.DownloadRequest{Token: first.Token, IPAddress: "127.0.0.1"})
	if err != nil {
		t.Fatalf("download by token: %v", err)
	}
	if res.DownloadsRemaining != 1 {
		t.Fatalf("expected downloads remaining 1, got %d", res.DownloadsRemaining)
	}
}

func TestRevokeLinksBlocksDownload(t *testing.T) {
	svc := newService()
	admin := application.Actor{SubjectID: "admin-1", Role: "admin", IdempotencyKey: "idem-file-2"}
	if _, err := svc.UpsertProductFile(context.Background(), admin, application.UpsertProductFileInput{ProductID: "prod-2", FileName: "file.bin", SizeBytes: 2048}); err != nil {
		t.Fatalf("upsert file: %v", err)
	}
	user := application.Actor{SubjectID: "user-2", Role: "creator", IdempotencyKey: "idem-link-2"}
	link, err := svc.GetDownloadLink(context.Background(), user, application.GetDownloadLinkInput{ProductID: "prod-2"})
	if err != nil {
		t.Fatalf("get link: %v", err)
	}
	adminRevoke := application.Actor{SubjectID: "support-1", Role: "support", IdempotencyKey: "idem-revoke-1"}
	if _, err := svc.RevokeLinks(context.Background(), adminRevoke, application.RevokeLinksInput{ProductID: "prod-2", UserID: "user-2", Reason: "refund"}); err != nil {
		t.Fatalf("revoke links: %v", err)
	}
	if _, err := svc.DownloadByToken(context.Background(), application.DownloadRequest{Token: link.Token, IPAddress: "127.0.0.2"}); err == nil {
		t.Fatalf("expected revoked token failure")
	}
}
