package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/viralforge/mesh/services/platform-ops/M83-cdn-management-service/internal/adapters/http"
	"github.com/viralforge/mesh/services/platform-ops/M83-cdn-management-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/platform-ops/M83-cdn-management-service/internal/application"
	"github.com/viralforge/mesh/services/platform-ops/M83-cdn-management-service/internal/contracts"
)

func newRouter() http.Handler {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Configs:      repos.Configs,
		Purges:       repos.Purges,
		Metrics:      repos.Metrics,
		Certificates: repos.Certificates,
		Idempotency:  repos.Idempotency,
	})
	return httpadapter.NewRouter(httpadapter.NewHandler(svc))
}

func TestCreateConfigRequiresIdempotencyKey(t *testing.T) {
	router := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/configs", strings.NewReader(`{"provider":"cloudflare","config":{"cache_ttl":300}}`))
	req.Header.Set("Authorization", "Bearer ops-1")
	req.Header.Set("X-Actor-Role", "ops_admin")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got=%d want=%d body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	var out contracts.ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if out.Code != "idempotency_key_required" || out.Error.Code != "idempotency_key_required" {
		t.Fatalf("unexpected error envelope: %+v", out)
	}
}

func TestCDNManagementRoutes(t *testing.T) {
	router := newRouter()

	healthReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthRR := httptest.NewRecorder()
	router.ServeHTTP(healthRR, healthReq)
	if healthRR.Code != http.StatusOK {
		t.Fatalf("health failed: status=%d body=%s", healthRR.Code, healthRR.Body.String())
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRR := httptest.NewRecorder()
	router.ServeHTTP(metricsRR, metricsReq)
	if metricsRR.Code != http.StatusOK {
		t.Fatalf("metrics failed: status=%d body=%s", metricsRR.Code, metricsRR.Body.String())
	}

	createReq := httptest.NewRequest(http.MethodPost, "/configs", strings.NewReader(`{"provider":"cloudflare","config":{"cache_ttl":600},"header_rules":{"Cache-Control":"public"}}`))
	createReq.Header.Set("Authorization", "Bearer ops-1")
	createReq.Header.Set("X-Actor-Role", "ops_admin")
	createReq.Header.Set("Idempotency-Key", "idem-create")
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("create config failed: status=%d body=%s", createRR.Code, createRR.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/configs", nil)
	listReq.Header.Set("Authorization", "Bearer ops-1")
	listReq.Header.Set("X-Actor-Role", "ops_admin")
	listRR := httptest.NewRecorder()
	router.ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("list configs failed: status=%d body=%s", listRR.Code, listRR.Body.String())
	}

	purgeReq := httptest.NewRequest(http.MethodPost, "/purge", strings.NewReader(`{"scope":"url","target":"https://cdn.example.com/assets/logo.png"}`))
	purgeReq.Header.Set("Authorization", "Bearer ops-1")
	purgeReq.Header.Set("X-Actor-Role", "ops_admin")
	purgeReq.Header.Set("Idempotency-Key", "idem-purge")
	purgeRR := httptest.NewRecorder()
	router.ServeHTTP(purgeRR, purgeReq)
	if purgeRR.Code != http.StatusCreated {
		t.Fatalf("purge failed: status=%d body=%s", purgeRR.Code, purgeRR.Body.String())
	}
}
