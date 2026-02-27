package unit

import (
	"context"
	"testing"

	"github.com/viralforge/mesh/services/integrations/M66-embed-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/integrations/M66-embed-service/internal/application"
)

func newService() *application.Service {
	repos := postgres.NewRepositories()
	return application.NewService(application.Dependencies{Settings: repos.Settings, Cache: repos.Cache, Impressions: repos.Impressions, Interactions: repos.Interactions, Idempotency: repos.Idempotency, EventDedup: repos.EventDedup})
}

func TestUpdateSettingsIdempotent(t *testing.T) {
	svc := newService()
	allow := true
	autoPlay := false
	showInfo := true
	actor := application.Actor{SubjectID: "creator-1", Role: "creator", IdempotencyKey: "idem-settings-1"}
	in := application.UpdateEmbedSettingsInput{EntityType: "campaign", EntityID: "ABC123", AllowEmbedding: &allow, DefaultTheme: "dark", PrimaryColor: "#6366F1", CustomButtonText: "Learn More", AutoPlayVideo: &autoPlay, ShowCreatorInfo: &showInfo, WhitelistedDomains: []string{"myblog.com"}}
	first, err := svc.UpdateSettings(context.Background(), actor, in)
	if err != nil {
		t.Fatalf("first update: %v", err)
	}
	second, err := svc.UpdateSettings(context.Background(), actor, in)
	if err != nil {
		t.Fatalf("second update: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected idempotent settings update")
	}
}

func TestRenderEmbedLogsImpressionAndAnalytics(t *testing.T) {
	svc := newService()
	_, err := svc.RenderEmbed(context.Background(), application.RenderEmbedInput{EntityType: "campaign", EntityID: "ABC123", Theme: "light", Color: "#5B21B6", ButtonText: "Join Now", Referrer: "https://myblog.com/article", UserAgent: "Mozilla/5.0 Chrome/120.0", ClientIP: "192.168.1.44"})
	if err != nil {
		t.Fatalf("render embed: %v", err)
	}
	analytics, err := svc.GetAnalytics(context.Background(), application.Actor{SubjectID: "creator-1", Role: "creator"}, application.AnalyticsQuery{EntityType: "campaign", EntityID: "ABC123", Granularity: "daily"})
	if err != nil {
		t.Fatalf("analytics: %v", err)
	}
	if analytics.TotalImpressions != 1 {
		t.Fatalf("expected 1 impression, got %d", analytics.TotalImpressions)
	}
}
