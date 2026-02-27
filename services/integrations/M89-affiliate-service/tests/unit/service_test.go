package unit

import (
	"context"
	"testing"

	eventadapter "github.com/viralforge/mesh/services/integrations/M89-affiliate-service/internal/adapters/events"
	"github.com/viralforge/mesh/services/integrations/M89-affiliate-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/integrations/M89-affiliate-service/internal/application"
)

func newServiceWithPublisher() (*application.Service, *eventadapter.MemoryDomainPublisher) {
	repos := postgres.NewRepositories()
	domainPub := eventadapter.NewMemoryDomainPublisher()
	svc := application.NewService(application.Dependencies{
		Affiliates: repos.Affiliates, Links: repos.Links, Clicks: repos.Clicks, Attributions: repos.Attributions,
		Earnings: repos.Earnings, Payouts: repos.Payouts, AuditLogs: repos.AuditLogs, Idempotency: repos.Idempotency,
		EventDedup: repos.EventDedup, Outbox: repos.Outbox, DomainEvents: domainPub,
	})
	return svc, domainPub
}

func TestCreateReferralLinkIdempotent(t *testing.T) {
	svc, _ := newServiceWithPublisher()
	actor := application.Actor{SubjectID: "user-1", Role: "affiliate", IdempotencyKey: "idem-link-1"}
	in := application.CreateReferralLinkInput{Channel: "youtube", UTMSource: "yt", UTMMedium: "video", UTMCampaign: "launch"}

	first, err := svc.CreateReferralLink(context.Background(), actor, in)
	if err != nil {
		t.Fatalf("first create link: %v", err)
	}
	second, err := svc.CreateReferralLink(context.Background(), actor, in)
	if err != nil {
		t.Fatalf("second create link: %v", err)
	}
	if first.LinkID != second.LinkID {
		t.Fatalf("expected idempotent link creation, got %s vs %s", first.LinkID, second.LinkID)
	}
}

func TestTrackClickFlushesCanonicalEvent(t *testing.T) {
	svc, pub := newServiceWithPublisher()
	ctx := context.Background()
	actor := application.Actor{SubjectID: "user-2", Role: "affiliate", IdempotencyKey: "idem-link-2"}
	link, err := svc.CreateReferralLink(ctx, actor, application.CreateReferralLinkInput{Channel: "twitter"})
	if err != nil {
		t.Fatalf("create link: %v", err)
	}
	_, err = svc.TrackReferralClick(ctx, application.TrackClickInput{Token: link.Token, ClientIP: "203.0.113.10", UserAgent: "Mozilla/5.0", ReferrerURL: "https://example.com"})
	if err != nil {
		t.Fatalf("track click: %v", err)
	}
	if err := svc.FlushOutbox(ctx); err != nil {
		t.Fatalf("flush outbox: %v", err)
	}
	if len(pub.Events) < 2 {
		t.Fatalf("expected at least 2 published events (link created + click), got %d", len(pub.Events))
	}
	found := false
	for _, e := range pub.Events {
		if e.EventType == "affiliate.click.tracked" {
			found = true
			if e.PartitionKeyPath != "data.affiliate_id" || e.PartitionKey == "" {
				t.Fatalf("partition key invariant not set")
			}
		}
	}
	if !found {
		t.Fatalf("click tracked event not found")
	}
	if pub.Events[0].PartitionKeyPath != "data.affiliate_id" && pub.Events[1].PartitionKeyPath != "data.affiliate_id" {
		t.Fatalf("partition key invariant not set")
	}
}

func TestRecordAttributionCreatesPendingEarning(t *testing.T) {
	svc, pub := newServiceWithPublisher()
	ctx := context.Background()
	affiliateActor := application.Actor{SubjectID: "user-3", Role: "affiliate", IdempotencyKey: "idem-link-3"}
	link, err := svc.CreateReferralLink(ctx, affiliateActor, application.CreateReferralLinkInput{Channel: "blog"})
	if err != nil {
		t.Fatalf("create link: %v", err)
	}
	click, err := svc.TrackReferralClick(ctx, application.TrackClickInput{Token: link.Token, ClientIP: "198.51.100.8", UserAgent: "UA", ReferrerURL: "https://ref.example"})
	if err != nil {
		t.Fatalf("track click: %v", err)
	}
	admin := application.Actor{SubjectID: "admin-1", Role: "admin", IdempotencyKey: "idem-attr-1"}
	_, err = svc.RecordAttribution(ctx, admin, application.RecordAttributionInput{
		AffiliateID:  click.AffiliateID,
		ClickID:      click.CookieID, // arbitrary linkage accepted by current service logic
		OrderID:      "ord-1",
		ConversionID: "conv-1",
		Amount:       200,
		Currency:     "USD",
	})
	if err != nil {
		t.Fatalf("record attribution: %v", err)
	}
	if err := svc.FlushOutbox(ctx); err != nil {
		t.Fatalf("flush outbox: %v", err)
	}
	if len(pub.Events) < 3 {
		t.Fatalf("expected events (link + click + attribution + earning), got %d", len(pub.Events))
	}
	dashboard, err := svc.GetDashboard(ctx, application.Actor{SubjectID: "user-3", Role: "affiliate"})
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if dashboard.TotalAttributions != 1 {
		t.Fatalf("expected 1 attribution, got %d", dashboard.TotalAttributions)
	}
	if dashboard.PendingEarnings <= 0 {
		t.Fatalf("expected pending earnings > 0, got %f", dashboard.PendingEarnings)
	}
}
