package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/viralforge/mesh/services/trust-compliance/M91-license-service/internal/adapters/http"
	"github.com/viralforge/mesh/services/trust-compliance/M91-license-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/trust-compliance/M91-license-service/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M91-license-service/internal/contracts"
)

func newRouter() http.Handler {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{Licenses: repos.Licenses, Activations: repos.Activations, Revocations: repos.Revocations, Configs: repos.Configs, Idempotency: repos.Idempotency})
	return httpadapter.NewRouter(httpadapter.NewHandler(svc))
}

func TestActivateRequiresIdempotencyKey(t *testing.T) {
	router := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/licenses/activate", strings.NewReader(`{"license_key":"ABCDE-FGHIJ-KLMNO-PQRST","device_id":"device-1","device_fingerprint":"fp-1"}`))
	req.Header.Set("Authorization", "Bearer user-1")
	req.Header.Set("X-Actor-Role", "user")
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

func TestLicenseRoutes(t *testing.T) {
	router := newRouter()

	listReq := httptest.NewRequest(http.MethodGet, "/licenses", nil)
	listReq.Header.Set("Authorization", "Bearer user-1")
	listReq.Header.Set("X-Actor-Role", "user")
	listRR := httptest.NewRecorder()
	router.ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("list licenses failed: status=%d body=%s", listRR.Code, listRR.Body.String())
	}

	validateReq := httptest.NewRequest(http.MethodGet, "/licenses/validate?license_key=ABCDE-FGHIJ-KLMNO-PQRST", nil)
	validateReq.Header.Set("Authorization", "Bearer user-1")
	validateReq.Header.Set("X-Actor-Role", "user")
	validateReq.RemoteAddr = "203.0.113.30:1234"
	validateRR := httptest.NewRecorder()
	router.ServeHTTP(validateRR, validateReq)
	if validateRR.Code != http.StatusOK {
		t.Fatalf("validate failed: status=%d body=%s", validateRR.Code, validateRR.Body.String())
	}

	aliasValidateReq := httptest.NewRequest(http.MethodGet, "/validate?license_key=ABCDE-FGHIJ-KLMNO-PQRST", nil)
	aliasValidateReq.Header.Set("Authorization", "Bearer user-1")
	aliasValidateReq.Header.Set("X-Actor-Role", "user")
	aliasValidateReq.RemoteAddr = "203.0.113.31:1234"
	aliasValidateRR := httptest.NewRecorder()
	router.ServeHTTP(aliasValidateRR, aliasValidateReq)
	if aliasValidateRR.Code != http.StatusOK {
		t.Fatalf("alias validate failed: status=%d body=%s", aliasValidateRR.Code, aliasValidateRR.Body.String())
	}

	activateReq := httptest.NewRequest(http.MethodPost, "/activate", strings.NewReader(`{"license_key":"ABCDE-FGHIJ-KLMNO-PQRST","device_id":"device-1","device_fingerprint":"fp-1"}`))
	activateReq.Header.Set("Authorization", "Bearer user-1")
	activateReq.Header.Set("X-Actor-Role", "user")
	activateReq.Header.Set("Idempotency-Key", "idem-activate")
	activateReq.RemoteAddr = "203.0.113.32:1234"
	activateRR := httptest.NewRecorder()
	router.ServeHTTP(activateRR, activateReq)
	if activateRR.Code != http.StatusOK {
		t.Fatalf("activate failed: status=%d body=%s", activateRR.Code, activateRR.Body.String())
	}

	deactivateReq := httptest.NewRequest(http.MethodPost, "/licenses/deactivate", strings.NewReader(`{"license_key":"ABCDE-FGHIJ-KLMNO-PQRST","device_id":"device-1"}`))
	deactivateReq.Header.Set("Authorization", "Bearer user-1")
	deactivateReq.Header.Set("X-Actor-Role", "user")
	deactivateReq.Header.Set("Idempotency-Key", "idem-deactivate")
	deactivateReq.RemoteAddr = "203.0.113.33:1234"
	deactivateRR := httptest.NewRecorder()
	router.ServeHTTP(deactivateRR, deactivateReq)
	if deactivateRR.Code != http.StatusOK {
		t.Fatalf("deactivate failed: status=%d body=%s", deactivateRR.Code, deactivateRR.Body.String())
	}

	exportReq := httptest.NewRequest(http.MethodPost, "/licenses/exports", strings.NewReader(`{"format":"json"}`))
	exportReq.Header.Set("Authorization", "Bearer user-1")
	exportReq.Header.Set("X-Actor-Role", "user")
	exportReq.Header.Set("Idempotency-Key", "idem-export")
	exportRR := httptest.NewRecorder()
	router.ServeHTTP(exportRR, exportReq)
	if exportRR.Code != http.StatusOK {
		t.Fatalf("export failed: status=%d body=%s", exportRR.Code, exportRR.Body.String())
	}
}
