package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/viralforge/mesh/services/trust-compliance/M51-data-portability-service/internal/adapters/http"
	"github.com/viralforge/mesh/services/trust-compliance/M51-data-portability-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/trust-compliance/M51-data-portability-service/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M51-data-portability-service/internal/contracts"
)

func newRouter() http.Handler {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Exports:     repos.ExportRequests,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})
	return httpadapter.NewRouter(httpadapter.NewHandler(svc))
}

func TestExportsCreateRequiresIdempotencyKey(t *testing.T) {
	router := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/v1/exports", strings.NewReader(`{"user_id":"u1","format":"json"}`))
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

func TestExportsCreateAndRead(t *testing.T) {
	router := newRouter()
	createReq := httptest.NewRequest(http.MethodPost, "/v1/exports", strings.NewReader(`{"user_id":"u2","format":"csv"}`))
	createReq.Header.Set("Authorization", "Bearer u2")
	createReq.Header.Set("Idempotency-Key", "idem-http-create")
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusOK {
		t.Fatalf("create failed: status=%d body=%s", createRR.Code, createRR.Body.String())
	}

	var createOut contracts.SuccessResponse
	if err := json.Unmarshal(createRR.Body.Bytes(), &createOut); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	dataBytes, _ := json.Marshal(createOut.Data)
	var row contracts.ExportRequestResponse
	if err := json.Unmarshal(dataBytes, &row); err != nil {
		t.Fatalf("decode export row: %v", err)
	}
	if row.RequestID == "" {
		t.Fatalf("missing request id: %+v", row)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/exports/"+row.RequestID, nil)
	getReq.Header.Set("Authorization", "Bearer u2")
	getRR := httptest.NewRecorder()
	router.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("get failed: status=%d body=%s", getRR.Code, getRR.Body.String())
	}
}

func TestExportsHistoryEndpoint(t *testing.T) {
	router := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/v1/exports", nil)
	req.Header.Set("Authorization", "Bearer u3")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("history failed: status=%d body=%s", rr.Code, rr.Body.String())
	}
}
