package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/viralforge/mesh/services/platform-ops/M84-data-migration-service/internal/adapters/http"
	"github.com/viralforge/mesh/services/platform-ops/M84-data-migration-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/platform-ops/M84-data-migration-service/internal/application"
	"github.com/viralforge/mesh/services/platform-ops/M84-data-migration-service/internal/contracts"
)

func newRouter() http.Handler {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{Plans: repos.Plans, Runs: repos.Runs, Registry: repos.Registry, Backfills: repos.Backfills, Metrics: repos.Metrics, Idempotency: repos.Idempotency})
	return httpadapter.NewRouter(httpadapter.NewHandler(svc))
}

func TestCreatePlanRequiresIdempotencyKey(t *testing.T) {
	router := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/plans", strings.NewReader(`{"service_name":"M01-Authentication-Service","environment":"staging","version":"2026.03.01","plan":{"steps":1}}`))
	req.Header.Set("Authorization", "Bearer ops-1")
	req.Header.Set("X-Actor-Role", "migration_operator")
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

func TestDataMigrationRoutes(t *testing.T) {
	router := newRouter()

	healthReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthRR := httptest.NewRecorder()
	router.ServeHTTP(healthRR, healthReq)
	if healthRR.Code != http.StatusOK {
		t.Fatalf("health failed: status=%d body=%s", healthRR.Code, healthRR.Body.String())
	}

	createReq := httptest.NewRequest(http.MethodPost, "/plans", strings.NewReader(`{"service_name":"M11-Distribution-Tracking-Service","environment":"staging","version":"2026.03.01","plan":{"changes":["add index"]},"risk_level":"low"}`))
	createReq.Header.Set("Authorization", "Bearer ops-1")
	createReq.Header.Set("X-Actor-Role", "migration_operator")
	createReq.Header.Set("Idempotency-Key", "idem-create")
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("create plan failed: status=%d body=%s", createRR.Code, createRR.Body.String())
	}
	var createOut contracts.SuccessResponse
	if err := json.Unmarshal(createRR.Body.Bytes(), &createOut); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	createData, _ := json.Marshal(createOut.Data)
	var plan struct {
		PlanID string `json:"plan_id"`
	}
	if err := json.Unmarshal(createData, &plan); err != nil {
		t.Fatalf("decode plan response: %v", err)
	}
	if plan.PlanID == "" {
		t.Fatalf("expected plan id in create response")
	}

	listReq := httptest.NewRequest(http.MethodGet, "/plans", nil)
	listReq.Header.Set("Authorization", "Bearer ops-1")
	listReq.Header.Set("X-Actor-Role", "migration_operator")
	listRR := httptest.NewRecorder()
	router.ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("list plans failed: status=%d body=%s", listRR.Code, listRR.Body.String())
	}

	runReq := httptest.NewRequest(http.MethodPost, "/runs", strings.NewReader(`{"plan_id":"`+plan.PlanID+`"}`))
	runReq.Header.Set("Authorization", "Bearer ops-1")
	runReq.Header.Set("X-Actor-Role", "migration_operator")
	runReq.Header.Set("X-MFA-Verified", "true")
	runReq.Header.Set("Idempotency-Key", "idem-run")
	runRR := httptest.NewRecorder()
	router.ServeHTTP(runRR, runReq)
	if runRR.Code != http.StatusCreated {
		t.Fatalf("create run failed: status=%d body=%s", runRR.Code, runRR.Body.String())
	}
}
