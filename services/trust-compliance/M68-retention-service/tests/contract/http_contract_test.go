package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/viralforge/mesh/services/trust-compliance/M68-retention-service/internal/adapters/http"
	"github.com/viralforge/mesh/services/trust-compliance/M68-retention-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/trust-compliance/M68-retention-service/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M68-retention-service/internal/contracts"
)

func newRouter() http.Handler {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Policies:     repos.Policies,
		Previews:     repos.Previews,
		Holds:        repos.Holds,
		Restorations: repos.Restorations,
		Deletions:    repos.Deletions,
		Audit:        repos.Audit,
		Idempotency:  repos.Idempotency,
	})
	return httpadapter.NewRouter(httpadapter.NewHandler(svc))
}

func TestCreatePolicyRequiresIdempotencyKey(t *testing.T) {
	router := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/retention/policies", strings.NewReader(`{"data_type":"messages","retention_years":7,"soft_delete_grace_days":30}`))
	req.Header.Set("Authorization", "Bearer admin-1")
	req.Header.Set("X-Actor-Role", "admin")
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

func TestRetentionRoutes(t *testing.T) {
	router := newRouter()

	policyReq := httptest.NewRequest(http.MethodPost, "/api/v1/retention/policies", strings.NewReader(`{"data_type":"messages","retention_years":5,"soft_delete_grace_days":14}`))
	policyReq.Header.Set("Authorization", "Bearer legal-1")
	policyReq.Header.Set("X-Actor-Role", "legal")
	policyReq.Header.Set("Idempotency-Key", "idem-http-policy")
	policyRR := httptest.NewRecorder()
	router.ServeHTTP(policyRR, policyReq)
	if policyRR.Code != http.StatusOK {
		t.Fatalf("create policy failed: status=%d body=%s", policyRR.Code, policyRR.Body.String())
	}

	var policyOut contracts.SuccessResponse
	if err := json.Unmarshal(policyRR.Body.Bytes(), &policyOut); err != nil {
		t.Fatalf("decode policy response: %v", err)
	}
	policyData, _ := json.Marshal(policyOut.Data)
	var policy contracts.RetentionPolicyResponse
	if err := json.Unmarshal(policyData, &policy); err != nil {
		t.Fatalf("decode policy data: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/retention/policies", nil)
	listReq.Header.Set("Authorization", "Bearer support-1")
	listReq.Header.Set("X-Actor-Role", "support")
	listRR := httptest.NewRecorder()
	router.ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("list policies failed: status=%d body=%s", listRR.Code, listRR.Body.String())
	}

	previewReq := httptest.NewRequest(http.MethodPost, "/api/v1/retention/preview", strings.NewReader(`{"policy_id":"`+policy.PolicyID+`"}`))
	previewReq.Header.Set("Authorization", "Bearer legal-1")
	previewReq.Header.Set("X-Actor-Role", "legal")
	previewRR := httptest.NewRecorder()
	router.ServeHTTP(previewRR, previewReq)
	if previewRR.Code != http.StatusOK {
		t.Fatalf("create preview failed: status=%d body=%s", previewRR.Code, previewRR.Body.String())
	}

	var previewOut contracts.SuccessResponse
	if err := json.Unmarshal(previewRR.Body.Bytes(), &previewOut); err != nil {
		t.Fatalf("decode preview response: %v", err)
	}
	previewData, _ := json.Marshal(previewOut.Data)
	var preview contracts.DeletionPreviewResponse
	if err := json.Unmarshal(previewData, &preview); err != nil {
		t.Fatalf("decode preview data: %v", err)
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/api/v1/retention/preview/"+preview.PreviewID+"/approve", strings.NewReader(`{"reason":"routine retention"}`))
	approveReq.Header.Set("Authorization", "Bearer legal-1")
	approveReq.Header.Set("X-Actor-Role", "legal")
	approveRR := httptest.NewRecorder()
	router.ServeHTTP(approveRR, approveReq)
	if approveRR.Code != http.StatusOK {
		t.Fatalf("approve preview failed: status=%d body=%s", approveRR.Code, approveRR.Body.String())
	}

	holdReq := httptest.NewRequest(http.MethodPost, "/api/v1/retention/legal-holds", strings.NewReader(`{"entity_id":"user-77","data_type":"payments","reason":"litigation"}`))
	holdReq.Header.Set("Authorization", "Bearer legal-1")
	holdReq.Header.Set("X-Actor-Role", "legal")
	holdReq.Header.Set("Idempotency-Key", "idem-http-hold")
	holdRR := httptest.NewRecorder()
	router.ServeHTTP(holdRR, holdReq)
	if holdRR.Code != http.StatusOK {
		t.Fatalf("create hold failed: status=%d body=%s", holdRR.Code, holdRR.Body.String())
	}

	restoreReq := httptest.NewRequest(http.MethodPost, "/api/v1/retention/restorations", strings.NewReader(`{"entity_id":"user-77","data_type":"payments","reason":"recovery"}`))
	restoreReq.Header.Set("Authorization", "Bearer admin-1")
	restoreReq.Header.Set("X-Actor-Role", "admin")
	restoreReq.Header.Set("Idempotency-Key", "idem-http-restore")
	restoreRR := httptest.NewRecorder()
	router.ServeHTTP(restoreRR, restoreReq)
	if restoreRR.Code != http.StatusOK {
		t.Fatalf("create restoration failed: status=%d body=%s", restoreRR.Code, restoreRR.Body.String())
	}

	var restoreOut contracts.SuccessResponse
	if err := json.Unmarshal(restoreRR.Body.Bytes(), &restoreOut); err != nil {
		t.Fatalf("decode restoration response: %v", err)
	}
	restoreData, _ := json.Marshal(restoreOut.Data)
	var restoration contracts.RestorationResponse
	if err := json.Unmarshal(restoreData, &restoration); err != nil {
		t.Fatalf("decode restoration data: %v", err)
	}

	approveRestoreReq := httptest.NewRequest(http.MethodPost, "/api/v1/retention/restorations/"+restoration.RestorationID+"/approve", strings.NewReader(`{"reason":"approved"}`))
	approveRestoreReq.Header.Set("Authorization", "Bearer admin-1")
	approveRestoreReq.Header.Set("X-Actor-Role", "admin")
	approveRestoreRR := httptest.NewRecorder()
	router.ServeHTTP(approveRestoreRR, approveRestoreReq)
	if approveRestoreRR.Code != http.StatusOK {
		t.Fatalf("approve restoration failed: status=%d body=%s", approveRestoreRR.Code, approveRestoreRR.Body.String())
	}

	reportReq := httptest.NewRequest(http.MethodGet, "/api/v1/retention/reports/compliance", nil)
	reportReq.Header.Set("Authorization", "Bearer support-1")
	reportReq.Header.Set("X-Actor-Role", "support")
	reportRR := httptest.NewRecorder()
	router.ServeHTTP(reportRR, reportReq)
	if reportRR.Code != http.StatusOK {
		t.Fatalf("report failed: status=%d body=%s", reportRR.Code, reportRR.Body.String())
	}
}
