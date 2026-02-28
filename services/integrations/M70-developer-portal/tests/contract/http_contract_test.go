package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/viralforge/mesh/services/integrations/M70-developer-portal/internal/adapters/http"
	"github.com/viralforge/mesh/services/integrations/M70-developer-portal/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/integrations/M70-developer-portal/internal/application"
	"github.com/viralforge/mesh/services/integrations/M70-developer-portal/internal/contracts"
)

func newRouter() http.Handler {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Developers:  repos.Developers,
		Sessions:    repos.Sessions,
		APIKeys:     repos.APIKeys,
		Rotations:   repos.Rotations,
		Webhooks:    repos.Webhooks,
		Deliveries:  repos.Deliveries,
		Usage:       repos.Usage,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})
	return httpadapter.NewRouter(httpadapter.NewHandler(svc))
}

func TestRegisterRequiresIdempotencyKey(t *testing.T) {
	router := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/developers/register", strings.NewReader(`{"email":"dev@example.com","app_name":"Portal"}`))
	req.Header.Set("Authorization", "Bearer requester-1")
	req.Header.Set("X-Actor-Role", "developer")
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

func TestDeveloperPortalRoutes(t *testing.T) {
	router := newRouter()

	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/developers/register", strings.NewReader(`{"email":"dev@example.com","app_name":"Portal"}`))
	registerReq.Header.Set("Authorization", "Bearer requester-1")
	registerReq.Header.Set("X-Actor-Role", "developer")
	registerReq.Header.Set("Idempotency-Key", "idem-http-register")
	registerRR := httptest.NewRecorder()
	router.ServeHTTP(registerRR, registerReq)
	if registerRR.Code != http.StatusOK {
		t.Fatalf("register failed: status=%d body=%s", registerRR.Code, registerRR.Body.String())
	}
	var registerOut contracts.SuccessResponse
	if err := json.Unmarshal(registerRR.Body.Bytes(), &registerOut); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	data, _ := json.Marshal(registerOut.Data)
	var reg contracts.RegisterDeveloperResponse
	if err := json.Unmarshal(data, &reg); err != nil {
		t.Fatalf("decode register data: %v", err)
	}

	keyReq := httptest.NewRequest(http.MethodPost, "/api/v1/developers/api-keys", strings.NewReader(`{"developer_id":"`+reg.Developer.DeveloperID+`","label":"Primary"}`))
	keyReq.Header.Set("Authorization", "Bearer "+reg.Developer.DeveloperID)
	keyReq.Header.Set("X-Actor-Role", "developer")
	keyReq.Header.Set("Idempotency-Key", "idem-http-key")
	keyRR := httptest.NewRecorder()
	router.ServeHTTP(keyRR, keyReq)
	if keyRR.Code != http.StatusOK {
		t.Fatalf("create key failed: status=%d body=%s", keyRR.Code, keyRR.Body.String())
	}
	var keyOut contracts.SuccessResponse
	if err := json.Unmarshal(keyRR.Body.Bytes(), &keyOut); err != nil {
		t.Fatalf("decode key response: %v", err)
	}
	keyData, _ := json.Marshal(keyOut.Data)
	var key contracts.APIKeyResponse
	if err := json.Unmarshal(keyData, &key); err != nil {
		t.Fatalf("decode key data: %v", err)
	}

	rotateReq := httptest.NewRequest(http.MethodPost, "/api/v1/developers/api-keys/"+key.KeyID+"/rotate", nil)
	rotateReq.Header.Set("Authorization", "Bearer "+reg.Developer.DeveloperID)
	rotateReq.Header.Set("X-Actor-Role", "developer")
	rotateReq.Header.Set("Idempotency-Key", "idem-http-rotate")
	rotateRR := httptest.NewRecorder()
	router.ServeHTTP(rotateRR, rotateReq)
	if rotateRR.Code != http.StatusOK {
		t.Fatalf("rotate key failed: status=%d body=%s", rotateRR.Code, rotateRR.Body.String())
	}
	var rotateOut contracts.SuccessResponse
	if err := json.Unmarshal(rotateRR.Body.Bytes(), &rotateOut); err != nil {
		t.Fatalf("decode rotate response: %v", err)
	}
	rotateData, _ := json.Marshal(rotateOut.Data)
	var rotation contracts.APIKeyRotationResponse
	if err := json.Unmarshal(rotateData, &rotation); err != nil {
		t.Fatalf("decode rotation: %v", err)
	}

	revokeReq := httptest.NewRequest(http.MethodPost, "/api/v1/developers/api-keys/"+rotation.NewKey.KeyID+"/revoke", nil)
	revokeReq.Header.Set("Authorization", "Bearer "+reg.Developer.DeveloperID)
	revokeReq.Header.Set("X-Actor-Role", "developer")
	revokeRR := httptest.NewRecorder()
	router.ServeHTTP(revokeRR, revokeReq)
	if revokeRR.Code != http.StatusOK {
		t.Fatalf("revoke key failed: status=%d body=%s", revokeRR.Code, revokeRR.Body.String())
	}

	webhookReq := httptest.NewRequest(http.MethodPost, "/webhooks", strings.NewReader(`{"developer_id":"`+reg.Developer.DeveloperID+`","url":"https://example.com/hook","event_type":"order.created"}`))
	webhookReq.Header.Set("Authorization", "Bearer "+reg.Developer.DeveloperID)
	webhookReq.Header.Set("X-Actor-Role", "developer")
	webhookReq.Header.Set("Idempotency-Key", "idem-http-webhook")
	webhookRR := httptest.NewRecorder()
	router.ServeHTTP(webhookRR, webhookReq)
	if webhookRR.Code != http.StatusOK {
		t.Fatalf("webhook create failed: status=%d body=%s", webhookRR.Code, webhookRR.Body.String())
	}
	var webhookOut contracts.SuccessResponse
	if err := json.Unmarshal(webhookRR.Body.Bytes(), &webhookOut); err != nil {
		t.Fatalf("decode webhook response: %v", err)
	}
	webhookData, _ := json.Marshal(webhookOut.Data)
	var webhook contracts.WebhookResponse
	if err := json.Unmarshal(webhookData, &webhook); err != nil {
		t.Fatalf("decode webhook data: %v", err)
	}

	testReq := httptest.NewRequest(http.MethodPost, "/api/v1/developers/webhooks/"+webhook.WebhookID+"/test", nil)
	testReq.Header.Set("Authorization", "Bearer "+reg.Developer.DeveloperID)
	testReq.Header.Set("X-Actor-Role", "developer")
	testRR := httptest.NewRecorder()
	router.ServeHTTP(testRR, testReq)
	if testRR.Code != http.StatusOK {
		t.Fatalf("webhook test failed: status=%d body=%s", testRR.Code, testRR.Body.String())
	}
}
