package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/viralforge/mesh/services/integrations/M73-support-service/internal/adapters/http"
	"github.com/viralforge/mesh/services/integrations/M73-support-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/integrations/M73-support-service/internal/application"
	"github.com/viralforge/mesh/services/integrations/M73-support-service/internal/contracts"
)

func newRouter() http.Handler {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Tickets:     repos.Tickets,
		Replies:     repos.Replies,
		CSAT:        repos.CSAT,
		Agents:      repos.Agents,
		Idempotency: repos.Idempotency,
	})
	return httpadapter.NewRouter(httpadapter.NewHandler(svc))
}

func TestCreateTicketRequiresIdempotencyKey(t *testing.T) {
	router := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/support/tickets", strings.NewReader(`{"subject":"Billing issue","description":"I need help with a payout issue.","category":"Billing"}`))
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

func TestSupportRoutes(t *testing.T) {
	router := newRouter()

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/support/tickets", strings.NewReader(`{"subject":"Refund request","description":"I need help with a refund overcharge on my order.","category":"Refund","priority":"high"}`))
	createReq.Header.Set("Authorization", "Bearer user-9")
	createReq.Header.Set("X-Actor-Role", "user")
	createReq.Header.Set("Idempotency-Key", "idem-create")
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("create ticket failed: status=%d body=%s", createRR.Code, createRR.Body.String())
	}
	var createOut contracts.SuccessResponse
	if err := json.Unmarshal(createRR.Body.Bytes(), &createOut); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	createData, _ := json.Marshal(createOut.Data)
	var ticket struct {
		TicketID string `json:"ticket_id"`
	}
	if err := json.Unmarshal(createData, &ticket); err != nil {
		t.Fatalf("decode ticket response: %v", err)
	}
	if ticket.TicketID == "" {
		t.Fatalf("expected ticket id in create response")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/support/tickets/"+ticket.TicketID, nil)
	getReq.Header.Set("Authorization", "Bearer user-9")
	getRR := httptest.NewRecorder()
	router.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("get ticket failed: status=%d body=%s", getRR.Code, getRR.Body.String())
	}

	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/support/tickets/"+ticket.TicketID, strings.NewReader(`{"status":"resolved"}`))
	patchReq.Header.Set("Authorization", "Bearer agent-technical")
	patchReq.Header.Set("X-Actor-Role", "agent")
	patchReq.Header.Set("Idempotency-Key", "idem-patch")
	patchRR := httptest.NewRecorder()
	router.ServeHTTP(patchRR, patchReq)
	if patchRR.Code != http.StatusOK {
		t.Fatalf("patch ticket failed: status=%d body=%s", patchRR.Code, patchRR.Body.String())
	}

	replyReq := httptest.NewRequest(http.MethodPost, "/api/v1/support/tickets/"+ticket.TicketID+"/replies", strings.NewReader(`{"reply_type":"public","body":"Please try the updated flow."}`))
	replyReq.Header.Set("Authorization", "Bearer agent-technical")
	replyReq.Header.Set("X-Actor-Role", "agent")
	replyReq.Header.Set("Idempotency-Key", "idem-reply")
	replyRR := httptest.NewRecorder()
	router.ServeHTTP(replyRR, replyReq)
	if replyRR.Code != http.StatusCreated {
		t.Fatalf("add reply failed: status=%d body=%s", replyRR.Code, replyRR.Body.String())
	}

	csatReq := httptest.NewRequest(http.MethodPost, "/api/v1/support/tickets/"+ticket.TicketID+"/csat", strings.NewReader(`{"rating":5,"feedback_comment":"Very helpful"}`))
	csatReq.Header.Set("Authorization", "Bearer user-9")
	csatReq.Header.Set("X-Actor-Role", "user")
	csatReq.Header.Set("Idempotency-Key", "idem-csat")
	csatRR := httptest.NewRecorder()
	router.ServeHTTP(csatRR, csatReq)
	if csatRR.Code != http.StatusCreated {
		t.Fatalf("submit csat failed: status=%d body=%s", csatRR.Code, csatRR.Body.String())
	}

	assignReq := httptest.NewRequest(http.MethodPost, "/api/v1/support/admin/tickets/"+ticket.TicketID+"/assign", strings.NewReader(`{"agent_id":"agent-senior"}`))
	assignReq.Header.Set("Authorization", "Bearer manager-1")
	assignReq.Header.Set("X-Actor-Role", "support_manager")
	assignReq.Header.Set("Idempotency-Key", "idem-assign")
	assignRR := httptest.NewRecorder()
	router.ServeHTTP(assignRR, assignReq)
	if assignRR.Code != http.StatusOK {
		t.Fatalf("assign ticket failed: status=%d body=%s", assignRR.Code, assignRR.Body.String())
	}

	searchReq := httptest.NewRequest(http.MethodGet, "/api/v1/support/tickets/search?q=refund", nil)
	searchReq.Header.Set("Authorization", "Bearer manager-1")
	searchReq.Header.Set("X-Actor-Role", "support_manager")
	searchRR := httptest.NewRecorder()
	router.ServeHTTP(searchRR, searchReq)
	if searchRR.Code != http.StatusOK {
		t.Fatalf("search tickets failed: status=%d body=%s", searchRR.Code, searchRR.Body.String())
	}

	emailReq := httptest.NewRequest(http.MethodPost, "/api/internal/tickets/create-from-email", strings.NewReader(`{"sender_email":"user@example.com","subject":"Email issue","description":"This ticket came from email intake."}`))
	emailReq.Header.Set("Authorization", "Bearer system-intake")
	emailReq.Header.Set("X-Actor-Role", "system")
	emailReq.Header.Set("Idempotency-Key", "idem-email")
	emailRR := httptest.NewRecorder()
	router.ServeHTTP(emailRR, emailReq)
	if emailRR.Code != http.StatusCreated {
		t.Fatalf("create from email failed: status=%d body=%s", emailRR.Code, emailRR.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/support/tickets/"+ticket.TicketID, nil)
	deleteReq.Header.Set("Authorization", "Bearer manager-1")
	deleteReq.Header.Set("X-Actor-Role", "support_manager")
	deleteRR := httptest.NewRecorder()
	router.ServeHTTP(deleteRR, deleteReq)
	if deleteRR.Code != http.StatusOK {
		t.Fatalf("delete ticket failed: status=%d body=%s", deleteRR.Code, deleteRR.Body.String())
	}
}
