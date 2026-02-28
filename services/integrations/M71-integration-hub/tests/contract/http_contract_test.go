package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/viralforge/mesh/services/integrations/M71-integration-hub/internal/adapters/http"
	"github.com/viralforge/mesh/services/integrations/M71-integration-hub/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/integrations/M71-integration-hub/internal/application"
	"github.com/viralforge/mesh/services/integrations/M71-integration-hub/internal/contracts"
)

func newRouter() http.Handler {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Integrations: repos.Integrations,
		Credentials:  repos.Credentials,
		Workflows:    repos.Workflows,
		Executions:   repos.Executions,
		Webhooks:     repos.Webhooks,
		Deliveries:   repos.Deliveries,
		Analytics:    repos.Analytics,
		Logs:         repos.Logs,
		Idempotency:  repos.Idempotency,
	})
	return httpadapter.NewRouter(httpadapter.NewHandler(svc))
}

func TestAuthorizeRequiresIdempotencyKey(t *testing.T) {
	router := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/Slack/authorize", strings.NewReader(`{"integration_name":"my-slack"}`))
	req.Header.Set("Authorization", "Bearer user-1")
	req.Header.Set("X-Actor-Role", "user")
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

func TestIntegrationHubRoutes(t *testing.T) {
	router := newRouter()

	authReq := httptest.NewRequest(http.MethodPost, "/integrations/Slack/authorize", strings.NewReader(`{"integration_name":"my-slack"}`))
	authReq.Header.Set("Authorization", "Bearer user-1")
	authReq.Header.Set("X-Actor-Role", "user")
	authReq.Header.Set("Idempotency-Key", "idem-http-auth")
	authRR := httptest.NewRecorder()
	router.ServeHTTP(authRR, authReq)
	if authRR.Code != http.StatusOK {
		t.Fatalf("authorize failed: status=%d body=%s", authRR.Code, authRR.Body.String())
	}
	var authOut contracts.SuccessResponse
	if err := json.Unmarshal(authRR.Body.Bytes(), &authOut); err != nil {
		t.Fatalf("decode auth response: %v", err)
	}
	authData, _ := json.Marshal(authOut.Data)
	var integration contracts.IntegrationResponse
	if err := json.Unmarshal(authData, &integration); err != nil {
		t.Fatalf("decode integration: %v", err)
	}

	workflowReq := httptest.NewRequest(http.MethodPost, "/workflows", strings.NewReader(`{"workflow_name":"Notify Slack","trigger_event_type":"submission.approved","action_type":"send_slack_message","integration_id":"`+integration.IntegrationID+`"}`))
	workflowReq.Header.Set("Authorization", "Bearer user-1")
	workflowReq.Header.Set("X-Actor-Role", "user")
	workflowReq.Header.Set("Idempotency-Key", "idem-http-workflow")
	workflowRR := httptest.NewRecorder()
	router.ServeHTTP(workflowRR, workflowReq)
	if workflowRR.Code != http.StatusOK {
		t.Fatalf("workflow create failed: status=%d body=%s", workflowRR.Code, workflowRR.Body.String())
	}
	var workflowOut contracts.SuccessResponse
	if err := json.Unmarshal(workflowRR.Body.Bytes(), &workflowOut); err != nil {
		t.Fatalf("decode workflow response: %v", err)
	}
	workflowData, _ := json.Marshal(workflowOut.Data)
	var workflow contracts.WorkflowResponse
	if err := json.Unmarshal(workflowData, &workflow); err != nil {
		t.Fatalf("decode workflow: %v", err)
	}

	publishReq := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflow.WorkflowID+"/publish", nil)
	publishReq.Header.Set("Authorization", "Bearer user-1")
	publishReq.Header.Set("X-Actor-Role", "user")
	publishReq.Header.Set("Idempotency-Key", "idem-http-publish")
	publishRR := httptest.NewRecorder()
	router.ServeHTTP(publishRR, publishReq)
	if publishRR.Code != http.StatusOK {
		t.Fatalf("workflow publish failed: status=%d body=%s", publishRR.Code, publishRR.Body.String())
	}

	testWorkflowReq := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflow.WorkflowID+"/test", nil)
	testWorkflowReq.Header.Set("Authorization", "Bearer user-1")
	testWorkflowReq.Header.Set("X-Actor-Role", "user")
	testWorkflowRR := httptest.NewRecorder()
	router.ServeHTTP(testWorkflowRR, testWorkflowReq)
	if testWorkflowRR.Code != http.StatusOK {
		t.Fatalf("workflow test failed: status=%d body=%s", testWorkflowRR.Code, testWorkflowRR.Body.String())
	}

	webhookReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks", strings.NewReader(`{"endpoint_url":"https://example.com/hook","event_type":"submission.created"}`))
	webhookReq.Header.Set("Authorization", "Bearer user-1")
	webhookReq.Header.Set("X-Actor-Role", "user")
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
		t.Fatalf("decode webhook: %v", err)
	}

	testWebhookReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/"+webhook.WebhookID+"/test", nil)
	testWebhookReq.Header.Set("Authorization", "Bearer user-1")
	testWebhookReq.Header.Set("X-Actor-Role", "user")
	testWebhookRR := httptest.NewRecorder()
	router.ServeHTTP(testWebhookRR, testWebhookReq)
	if testWebhookRR.Code != http.StatusOK {
		t.Fatalf("webhook test failed: status=%d body=%s", testWebhookRR.Code, testWebhookRR.Body.String())
	}

	chatReq := httptest.NewRequest(http.MethodPost, "/chat.postMessage?channel=%23alerts", nil)
	chatReq.Header.Set("Authorization", "Bearer user-1")
	chatReq.Header.Set("X-Actor-Role", "user")
	chatRR := httptest.NewRecorder()
	router.ServeHTTP(chatRR, chatReq)
	if chatRR.Code != http.StatusOK {
		t.Fatalf("chat post message failed: status=%d body=%s", chatRR.Code, chatRR.Body.String())
	}
}
