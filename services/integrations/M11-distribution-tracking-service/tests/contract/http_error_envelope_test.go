package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/viralforge/mesh/services/integrations/M11-distribution-tracking-service/internal/adapters/http"
	"github.com/viralforge/mesh/services/integrations/M11-distribution-tracking-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/integrations/M11-distribution-tracking-service/internal/application"
	"github.com/viralforge/mesh/services/integrations/M11-distribution-tracking-service/internal/contracts"
)

func TestErrorEnvelopeMaintainsCanonicalAndNestedFields(t *testing.T) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Posts:       repos.Posts,
		Snapshots:   repos.Snapshots,
		Idempotency: repos.Idempotency,
		EventDedup:  repos.EventDedup,
		Outbox:      repos.Outbox,
	})
	router := httpadapter.NewRouter(httpadapter.NewHandler(svc))

	req := httptest.NewRequest(http.MethodPost, "/v1/tracking/posts/validate", strings.NewReader("{"))
	req.Header.Set("Authorization", "Bearer u1")
	req.Header.Set("Idempotency-Key", "idem-1")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got=%d want=%d", rr.Code, http.StatusBadRequest)
	}
	var resp contracts.ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != "error" || resp.Code != "invalid_input" || resp.Error.Code != "invalid_input" {
		t.Fatalf("unexpected error envelope: %+v", resp)
	}
	if resp.RequestID == "" || resp.Error.RequestID == "" || resp.RequestID != resp.Error.RequestID {
		t.Fatalf("request_id mismatch: top=%q nested=%q", resp.RequestID, resp.Error.RequestID)
	}
	if resp.Message == "" || resp.Error.Message == "" {
		t.Fatalf("error message missing: %+v", resp)
	}
}
