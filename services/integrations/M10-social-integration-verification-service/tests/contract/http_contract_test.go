package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	eventadapter "github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/adapters/events"
	httpadapter "github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/adapters/http"
	"github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/application"
)

func newRouter() http.Handler {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Accounts: repos.Accounts, Metrics: repos.Metrics, Idempotency: repos.Idempotency, EventDedup: repos.EventDedup, Outbox: repos.Outbox,
		DomainEvents: eventadapter.NewMemoryDomainPublisher(), Analytics: eventadapter.NewMemoryAnalyticsPublisher(), DLQ: eventadapter.NewLoggingDLQPublisher(),
	})
	return httpadapter.NewRouter(httpadapter.NewHandler(svc))
}

func TestConnectRouteSupportsVersionedAndUnversionedPaths(t *testing.T) {
	t.Parallel()
	router := newRouter()

	paths := []string{
		"/social/connect/instagram",
		"/v1/social/connect/instagram",
	}
	for i, path := range paths {
		body := map[string]any{}
		req := jsonRequest(t, path, body)
		req.Header.Set("Authorization", "Bearer user_1")
		req.Header.Set("Idempotency-Key", "idem-connect-route-"+string(rune('a'+i)))
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("path %s expected 200, got %d", path, res.Code)
		}
	}
}

func TestCallbackResponseIncludesProvider(t *testing.T) {
	t.Parallel()
	router := newRouter()

	req := jsonRequest(t, "/social/callback/instagram", map[string]any{
		"code":   "oauth-code",
		"state":  "state-1",
		"handle": "creator",
	})
	req.Header.Set("Authorization", "Bearer user_1")
	req.Header.Set("Idempotency-Key", "idem-callback-1")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data, _ := payload["data"].(map[string]any)
	if got, _ := data["provider"].(string); got != "instagram" {
		t.Fatalf("expected provider=instagram, got %v", data["provider"])
	}
}

func TestErrorEnvelopeIncludesTopLevelCodeAndNestedError(t *testing.T) {
	t.Parallel()
	router := newRouter()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/social/connect/instagram", bytes.NewBufferString("{"))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer user_1")
	req.Header.Set("Idempotency-Key", "idem-invalid-json")
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, _ := payload["code"].(string); got != "invalid_input" {
		t.Fatalf("expected top-level code invalid_input, got %v", payload["code"])
	}
	errObj, _ := payload["error"].(map[string]any)
	if got, _ := errObj["code"].(string); got != "invalid_input" {
		t.Fatalf("expected nested error.code invalid_input, got %v", errObj["code"])
	}
}

func jsonRequest(t *testing.T, path string, body any) *http.Request {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}
