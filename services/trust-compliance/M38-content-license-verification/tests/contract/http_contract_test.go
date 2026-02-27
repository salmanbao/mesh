package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/viralforge/mesh/services/trust-compliance/M38-content-license-verification/internal/adapters/http"
	"github.com/viralforge/mesh/services/trust-compliance/M38-content-license-verification/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/trust-compliance/M38-content-license-verification/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M38-content-license-verification/internal/contracts"
)

func newRouter() http.Handler {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Matches:     repos.Matches,
		Holds:       repos.Holds,
		Appeals:     repos.Appeals,
		Takedowns:   repos.Takedowns,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})
	return httpadapter.NewRouter(httpadapter.NewHandler(svc))
}

func TestScanRequiresIdempotencyKey(t *testing.T) {
	router := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/license/scan", strings.NewReader(`{"submission_id":"sub-1","creator_id":"u1","media_type":"video","media_url":"https://example.com/match.mp4"}`))
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
		t.Fatalf("unexpected error code: %+v", out)
	}
	if out.RequestID == "" || out.Error.RequestID == "" {
		t.Fatalf("missing request ids: %+v", out)
	}
}

func TestDMCARequiresAdminOrLegalRole(t *testing.T) {
	router := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/dmca-takedown", strings.NewReader(`{"submission_id":"sub-2","rights_holder_name":"Example Records","contact_email":"legal@example.com","reference":"DMCA-1"}`))
	req.Header.Set("Authorization", "Bearer user-1")
	req.Header.Set("Idempotency-Key", "idem-dmca-role")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("unexpected status: got=%d want=%d", rr.Code, http.StatusForbidden)
	}
}

func TestScanSuccessEnvelope(t *testing.T) {
	router := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/license/scan", strings.NewReader(`{"submission_id":"sub-3","creator_id":"u3","media_type":"audio","media_url":"https://example.com/copyrighted.mp3"}`))
	req.Header.Set("Authorization", "Bearer u3")
	req.Header.Set("Idempotency-Key", "idem-scan-ok")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var out contracts.SuccessResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode success response: %v", err)
	}
	if out.Status != "success" {
		t.Fatalf("unexpected success envelope: %+v", out)
	}
}
