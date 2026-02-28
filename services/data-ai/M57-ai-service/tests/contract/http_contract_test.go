package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/viralforge/mesh/services/data-ai/M57-ai-service/internal/adapters/http"
	"github.com/viralforge/mesh/services/data-ai/M57-ai-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/data-ai/M57-ai-service/internal/application"
	"github.com/viralforge/mesh/services/data-ai/M57-ai-service/internal/contracts"
)

func newRouter() http.Handler {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Predictions: repos.Predictions,
		BatchJobs:   repos.BatchJobs,
		Models:      repos.Models,
		Feedback:    repos.Feedback,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})
	return httpadapter.NewRouter(httpadapter.NewHandler(svc))
}

func TestAnalyzeRequiresIdempotencyKey(t *testing.T) {
	router := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/analyze", strings.NewReader(`{"content":"safe"}`))
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

func TestAnalyzeAndBatchStatusEndpoints(t *testing.T) {
	router := newRouter()

	analyzeReq := httptest.NewRequest(http.MethodPost, "/api/v1/ai/analyze", strings.NewReader(`{"content_id":"c1","content":"copyright claim notice"}`))
	analyzeReq.Header.Set("Authorization", "Bearer u2")
	analyzeReq.Header.Set("Idempotency-Key", "idem-http-analyze")
	analyzeRR := httptest.NewRecorder()
	router.ServeHTTP(analyzeRR, analyzeReq)
	if analyzeRR.Code != http.StatusOK {
		t.Fatalf("analyze failed: status=%d body=%s", analyzeRR.Code, analyzeRR.Body.String())
	}

	batchReq := httptest.NewRequest(http.MethodPost, "/api/v1/ai/batch-analyze", strings.NewReader(`{"items":[{"content_id":"c2","content":"safe"},{"content_id":"c3","content":"fraud warning"}]}`))
	batchReq.Header.Set("Authorization", "Bearer u2")
	batchReq.Header.Set("Idempotency-Key", "idem-http-batch")
	batchRR := httptest.NewRecorder()
	router.ServeHTTP(batchRR, batchReq)
	if batchRR.Code != http.StatusOK {
		t.Fatalf("batch failed: status=%d body=%s", batchRR.Code, batchRR.Body.String())
	}

	var batchOut contracts.SuccessResponse
	if err := json.Unmarshal(batchRR.Body.Bytes(), &batchOut); err != nil {
		t.Fatalf("decode batch response: %v", err)
	}
	dataBytes, _ := json.Marshal(batchOut.Data)
	var job contracts.BatchStatusResponse
	if err := json.Unmarshal(dataBytes, &job); err != nil {
		t.Fatalf("decode batch job: %v", err)
	}
	if job.JobID == "" {
		t.Fatalf("missing job id: %+v", job)
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/ai/batch-status/"+job.JobID, nil)
	statusReq.Header.Set("Authorization", "Bearer u2")
	statusRR := httptest.NewRecorder()
	router.ServeHTTP(statusRR, statusReq)
	if statusRR.Code != http.StatusOK {
		t.Fatalf("status failed: status=%d body=%s", statusRR.Code, statusRR.Body.String())
	}
}
