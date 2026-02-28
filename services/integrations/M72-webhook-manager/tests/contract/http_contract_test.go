package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/viralforge/mesh/services/integrations/M72-webhook-manager/internal/adapters/http"
	"github.com/viralforge/mesh/services/integrations/M72-webhook-manager/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/integrations/M72-webhook-manager/internal/application"
	"github.com/viralforge/mesh/services/integrations/M72-webhook-manager/internal/contracts"
)

func newRouter() http.Handler {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Webhooks:    repos.Webhooks,
		Deliveries:  repos.Deliveries,
		Analytics:   repos.Analytics,
		Idempotency: repos.Idempotency,
	})
	return httpadapter.NewRouter(httpadapter.NewHandler(svc))
}

func TestCreateRequiresIdempotencyKey(t *testing.T) {
	router := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks", strings.NewReader(`{"endpoint_url":"https://example.com/hook","event_types":["submission.created"]}`))
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

func TestWebhookManagerRoutes(t *testing.T) {
	router := newRouter()

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks", strings.NewReader(`{"endpoint_url":"https://example.com/hook","event_types":["submission.created"],"batch_size":5}`))
	createReq.Header.Set("Authorization", "Bearer user-1")
	createReq.Header.Set("X-Actor-Role", "user")
	createReq.Header.Set("Idempotency-Key", "idem-create")
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("create failed: status=%d body=%s", createRR.Code, createRR.Body.String())
	}
	var createOut contracts.SuccessResponse
	if err := json.Unmarshal(createRR.Body.Bytes(), &createOut); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	createData, _ := json.Marshal(createOut.Data)
	var created struct {
		WebhookID string `json:"webhook_id"`
		Status    string `json:"status"`
	}
	if err := json.Unmarshal(createData, &created); err != nil {
		t.Fatalf("decode created webhook: %v", err)
	}
	if created.WebhookID == "" || created.Status != "active" {
		t.Fatalf("unexpected create payload: %+v", created)
	}

	updateReq := httptest.NewRequest(http.MethodPatch, "/api/v1/webhooks/"+created.WebhookID, strings.NewReader(`{"status":"disabled","rate_limit_per_minute":20}`))
	updateReq.Header.Set("Authorization", "Bearer user-1")
	updateReq.Header.Set("X-Actor-Role", "user")
	updateReq.Header.Set("Idempotency-Key", "idem-update")
	updateRR := httptest.NewRecorder()
	router.ServeHTTP(updateRR, updateReq)
	if updateRR.Code != http.StatusOK {
		t.Fatalf("update failed: status=%d body=%s", updateRR.Code, updateRR.Body.String())
	}

	enableReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/"+created.WebhookID+"/enable", nil)
	enableReq.Header.Set("Authorization", "Bearer user-1")
	enableReq.Header.Set("X-Actor-Role", "user")
	enableReq.Header.Set("Idempotency-Key", "idem-enable")
	enableRR := httptest.NewRecorder()
	router.ServeHTTP(enableRR, enableReq)
	if enableRR.Code != http.StatusOK {
		t.Fatalf("enable failed: status=%d body=%s", enableRR.Code, enableRR.Body.String())
	}

	testReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/"+created.WebhookID+"/test", strings.NewReader(`{"payload":{"ok":true}}`))
	testReq.Header.Set("Authorization", "Bearer user-1")
	testReq.Header.Set("X-Actor-Role", "user")
	testReq.Header.Set("Idempotency-Key", "idem-test")
	testRR := httptest.NewRecorder()
	router.ServeHTTP(testRR, testReq)
	if testRR.Code != http.StatusOK {
		t.Fatalf("test failed: status=%d body=%s", testRR.Code, testRR.Body.String())
	}

	deliveriesReq := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/"+created.WebhookID+"/deliveries", nil)
	deliveriesReq.Header.Set("Authorization", "Bearer user-1")
	deliveriesReq.Header.Set("X-Actor-Role", "user")
	deliveriesRR := httptest.NewRecorder()
	router.ServeHTTP(deliveriesRR, deliveriesReq)
	if deliveriesRR.Code != http.StatusOK {
		t.Fatalf("deliveries failed: status=%d body=%s", deliveriesRR.Code, deliveriesRR.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/webhooks/"+created.WebhookID, nil)
	deleteReq.Header.Set("Authorization", "Bearer user-1")
	deleteReq.Header.Set("X-Actor-Role", "user")
	deleteRR := httptest.NewRecorder()
	router.ServeHTTP(deleteRR, deleteReq)
	if deleteRR.Code != http.StatusOK {
		t.Fatalf("delete failed: status=%d body=%s", deleteRR.Code, deleteRR.Body.String())
	}

	compatReq := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"event_id":"evt-1"}`))
	compatRR := httptest.NewRecorder()
	router.ServeHTTP(compatRR, compatReq)
	if compatRR.Code != http.StatusAccepted {
		t.Fatalf("compatibility webhook failed: status=%d body=%s", compatRR.Code, compatRR.Body.String())
	}
}
