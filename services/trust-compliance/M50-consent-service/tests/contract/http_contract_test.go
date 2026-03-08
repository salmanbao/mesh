package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/viralforge/mesh/services/trust-compliance/M50-consent-service/internal/adapters/http"
	"github.com/viralforge/mesh/services/trust-compliance/M50-consent-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/trust-compliance/M50-consent-service/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M50-consent-service/internal/contracts"
)

func newRouter() http.Handler {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Consents:    repos.Consents,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})
	return httpadapter.NewRouter(httpadapter.NewHandler(svc))
}

func TestConsentUpdateRequiresIdempotencyKey(t *testing.T) {
	router := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/v1/consent/u1/update", strings.NewReader(`{"preferences":{"analytics":true},"reason":"test"}`))
	req.Header.Set("Authorization", "Bearer u1")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got=%d want=%d", rr.Code, http.StatusBadRequest)
	}
	var out contracts.ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if out.Code != "idempotency_key_required" || out.Error.Code != "idempotency_key_required" {
		t.Fatalf("unexpected error envelope: %+v", out)
	}
}

func TestConsentUpdateGetAndHistory(t *testing.T) {
	router := newRouter()

	updateReq := httptest.NewRequest(http.MethodPost, "/v1/consent/u2/update", strings.NewReader(`{"preferences":{"analytics":true,"marketing":true},"reason":"initial"}`))
	updateReq.Header.Set("Authorization", "Bearer u2")
	updateReq.Header.Set("Idempotency-Key", "idem-u2-update")
	updateRR := httptest.NewRecorder()
	router.ServeHTTP(updateRR, updateReq)
	if updateRR.Code != http.StatusOK {
		t.Fatalf("update failed: status=%d body=%s", updateRR.Code, updateRR.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/consent/u2", nil)
	getReq.Header.Set("Authorization", "Bearer u2")
	getRR := httptest.NewRecorder()
	router.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("get failed: status=%d body=%s", getRR.Code, getRR.Body.String())
	}

	historyReq := httptest.NewRequest(http.MethodGet, "/v1/consent/u2/history?limit=5", nil)
	historyReq.Header.Set("Authorization", "Bearer u2")
	historyRR := httptest.NewRecorder()
	router.ServeHTTP(historyRR, historyReq)
	if historyRR.Code != http.StatusOK {
		t.Fatalf("history failed: status=%d body=%s", historyRR.Code, historyRR.Body.String())
	}
}

func TestAdminConsentRoutesAndPermissions(t *testing.T) {
	router := newRouter()

	adminUpdate := httptest.NewRequest(http.MethodPost, "/v1/admin/consent/target-1/update", strings.NewReader(`{"preferences":{"marketing":true},"reason":"policy update"}`))
	adminUpdate.Header.Set("Authorization", "Bearer admin-1")
	adminUpdate.Header.Set("X-Actor-Role", "admin")
	adminUpdate.Header.Set("Idempotency-Key", "idem-admin-update")
	adminUpdateRR := httptest.NewRecorder()
	router.ServeHTTP(adminUpdateRR, adminUpdate)
	if adminUpdateRR.Code != http.StatusOK {
		t.Fatalf("admin update failed: status=%d body=%s", adminUpdateRR.Code, adminUpdateRR.Body.String())
	}

	adminGet := httptest.NewRequest(http.MethodGet, "/v1/admin/consent/target-1", nil)
	adminGet.Header.Set("Authorization", "Bearer admin-1")
	adminGet.Header.Set("X-Actor-Role", "admin")
	adminGetRR := httptest.NewRecorder()
	router.ServeHTTP(adminGetRR, adminGet)
	if adminGetRR.Code != http.StatusOK {
		t.Fatalf("admin get failed: status=%d body=%s", adminGetRR.Code, adminGetRR.Body.String())
	}

	supportUpdate := httptest.NewRequest(http.MethodPost, "/v1/admin/consent/target-1/update", strings.NewReader(`{"preferences":{"marketing":false},"reason":"not allowed"}`))
	supportUpdate.Header.Set("Authorization", "Bearer support-1")
	supportUpdate.Header.Set("X-Actor-Role", "support")
	supportUpdate.Header.Set("Idempotency-Key", "idem-support-update")
	supportUpdateRR := httptest.NewRecorder()
	router.ServeHTTP(supportUpdateRR, supportUpdate)
	if supportUpdateRR.Code != http.StatusForbidden {
		t.Fatalf("support update should be forbidden: status=%d body=%s", supportUpdateRR.Code, supportUpdateRR.Body.String())
	}
}
