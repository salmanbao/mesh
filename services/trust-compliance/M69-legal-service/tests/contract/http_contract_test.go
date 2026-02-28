package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/viralforge/mesh/services/trust-compliance/M69-legal-service/internal/adapters/http"
	"github.com/viralforge/mesh/services/trust-compliance/M69-legal-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/trust-compliance/M69-legal-service/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M69-legal-service/internal/contracts"
)

func newRouter() http.Handler {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Documents:   repos.Documents,
		Signatures:  repos.Signatures,
		Holds:       repos.Holds,
		Compliance:  repos.Compliance,
		Disputes:    repos.Disputes,
		DMCA:        repos.DMCANotices,
		Filings:     repos.Filings,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})
	return httpadapter.NewRouter(httpadapter.NewHandler(svc))
}

func TestDocumentUploadRequiresIdempotencyKey(t *testing.T) {
	router := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/legal/documents/upload", strings.NewReader(`{"document_type":"terms","file_name":"tos.pdf"}`))
	req.Header.Set("Authorization", "Bearer legal-1")
	req.Header.Set("X-Actor-Role", "legal")
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

func TestLegalRoutes(t *testing.T) {
	router := newRouter()

	uploadReq := httptest.NewRequest(http.MethodPost, "/api/v1/legal/documents/upload", strings.NewReader(`{"document_type":"terms","file_name":"tos.pdf"}`))
	uploadReq.Header.Set("Authorization", "Bearer legal-1")
	uploadReq.Header.Set("X-Actor-Role", "legal")
	uploadReq.Header.Set("Idempotency-Key", "idem-http-doc")
	uploadRR := httptest.NewRecorder()
	router.ServeHTTP(uploadRR, uploadReq)
	if uploadRR.Code != http.StatusOK {
		t.Fatalf("upload failed: status=%d body=%s", uploadRR.Code, uploadRR.Body.String())
	}
	var uploadOut contracts.SuccessResponse
	if err := json.Unmarshal(uploadRR.Body.Bytes(), &uploadOut); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}
	uploadData, _ := json.Marshal(uploadOut.Data)
	var doc contracts.LegalDocumentResponse
	if err := json.Unmarshal(uploadData, &doc); err != nil {
		t.Fatalf("decode document: %v", err)
	}

	signReq := httptest.NewRequest(http.MethodPost, "/api/v1/legal/documents/"+doc.DocumentID+"/signatures", strings.NewReader(`{"signer_user_id":"user-11"}`))
	signReq.Header.Set("Authorization", "Bearer legal-1")
	signReq.Header.Set("X-Actor-Role", "legal")
	signRR := httptest.NewRecorder()
	router.ServeHTTP(signRR, signReq)
	if signRR.Code != http.StatusOK {
		t.Fatalf("signature request failed: status=%d body=%s", signRR.Code, signRR.Body.String())
	}

	holdReq := httptest.NewRequest(http.MethodPost, "/api/v1/legal/holds", strings.NewReader(`{"entity_type":"user","entity_id":"user-77","reason":"litigation"}`))
	holdReq.Header.Set("Authorization", "Bearer legal-1")
	holdReq.Header.Set("X-Actor-Role", "legal")
	holdReq.Header.Set("Idempotency-Key", "idem-http-hold")
	holdRR := httptest.NewRecorder()
	router.ServeHTTP(holdRR, holdReq)
	if holdRR.Code != http.StatusOK {
		t.Fatalf("hold failed: status=%d body=%s", holdRR.Code, holdRR.Body.String())
	}
	var holdOut contracts.SuccessResponse
	if err := json.Unmarshal(holdRR.Body.Bytes(), &holdOut); err != nil {
		t.Fatalf("decode hold response: %v", err)
	}
	holdData, _ := json.Marshal(holdOut.Data)
	var hold contracts.HoldResponse
	if err := json.Unmarshal(holdData, &hold); err != nil {
		t.Fatalf("decode hold: %v", err)
	}

	checkReq := httptest.NewRequest(http.MethodGet, "/api/v1/legal/holds/check?entity_type=user&entity_id=user-77", nil)
	checkReq.Header.Set("Authorization", "Bearer retention-service")
	checkReq.Header.Set("X-Actor-Role", "service")
	checkRR := httptest.NewRecorder()
	router.ServeHTTP(checkRR, checkReq)
	if checkRR.Code != http.StatusOK {
		t.Fatalf("hold check failed: status=%d body=%s", checkRR.Code, checkRR.Body.String())
	}

	releaseReq := httptest.NewRequest(http.MethodPost, "/api/v1/legal/holds/"+hold.HoldID+"/release", strings.NewReader(`{"reason":"released"}`))
	releaseReq.Header.Set("Authorization", "Bearer legal-1")
	releaseReq.Header.Set("X-Actor-Role", "legal")
	releaseRR := httptest.NewRecorder()
	router.ServeHTTP(releaseRR, releaseReq)
	if releaseRR.Code != http.StatusOK {
		t.Fatalf("hold release failed: status=%d body=%s", releaseRR.Code, releaseRR.Body.String())
	}

	scanReq := httptest.NewRequest(http.MethodPost, "/legal/compliance/scan", strings.NewReader(`{"report_type":"manual"}`))
	scanReq.Header.Set("Authorization", "Bearer legal-1")
	scanReq.Header.Set("X-Actor-Role", "legal")
	scanRR := httptest.NewRecorder()
	router.ServeHTTP(scanRR, scanReq)
	if scanRR.Code != http.StatusOK {
		t.Fatalf("compliance scan failed: status=%d body=%s", scanRR.Code, scanRR.Body.String())
	}
	var scanOut contracts.SuccessResponse
	if err := json.Unmarshal(scanRR.Body.Bytes(), &scanOut); err != nil {
		t.Fatalf("decode scan response: %v", err)
	}
	scanData, _ := json.Marshal(scanOut.Data)
	var report contracts.ComplianceReportResponse
	if err := json.Unmarshal(scanData, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}

	reportReq := httptest.NewRequest(http.MethodGet, "/api/v1/legal/compliance/reports/"+report.ReportID, nil)
	reportReq.Header.Set("Authorization", "Bearer support-1")
	reportReq.Header.Set("X-Actor-Role", "support")
	reportRR := httptest.NewRecorder()
	router.ServeHTTP(reportRR, reportReq)
	if reportRR.Code != http.StatusOK {
		t.Fatalf("get report failed: status=%d body=%s", reportRR.Code, reportRR.Body.String())
	}

	disputeReq := httptest.NewRequest(http.MethodPost, "/api/v1/legal/disputes", strings.NewReader(`{"user_id":"user-9","opposing_party":"seller-1","dispute_reason":"payment_dispute","amount_cents":1999}`))
	disputeReq.Header.Set("Authorization", "Bearer user-9")
	disputeReq.Header.Set("Idempotency-Key", "idem-http-dispute")
	disputeRR := httptest.NewRecorder()
	router.ServeHTTP(disputeRR, disputeReq)
	if disputeRR.Code != http.StatusOK {
		t.Fatalf("create dispute failed: status=%d body=%s", disputeRR.Code, disputeRR.Body.String())
	}
	var disputeOut contracts.SuccessResponse
	if err := json.Unmarshal(disputeRR.Body.Bytes(), &disputeOut); err != nil {
		t.Fatalf("decode dispute response: %v", err)
	}
	disputeData, _ := json.Marshal(disputeOut.Data)
	var dispute contracts.DisputeResponse
	if err := json.Unmarshal(disputeData, &dispute); err != nil {
		t.Fatalf("decode dispute: %v", err)
	}

	getDisputeReq := httptest.NewRequest(http.MethodGet, "/api/v1/legal/disputes/"+dispute.DisputeID, nil)
	getDisputeReq.Header.Set("Authorization", "Bearer support-1")
	getDisputeReq.Header.Set("X-Actor-Role", "support")
	getDisputeRR := httptest.NewRecorder()
	router.ServeHTTP(getDisputeRR, getDisputeReq)
	if getDisputeRR.Code != http.StatusOK {
		t.Fatalf("get dispute failed: status=%d body=%s", getDisputeRR.Code, getDisputeRR.Body.String())
	}

	dmcaReq := httptest.NewRequest(http.MethodPost, "/api/v1/legal/dmca-notices", strings.NewReader(`{"content_id":"content-1","claimant":"Studio","reason":"copyright infringement"}`))
	dmcaReq.Header.Set("Authorization", "Bearer legal-1")
	dmcaReq.Header.Set("X-Actor-Role", "legal")
	dmcaReq.Header.Set("Idempotency-Key", "idem-http-dmca")
	dmcaRR := httptest.NewRecorder()
	router.ServeHTTP(dmcaRR, dmcaReq)
	if dmcaRR.Code != http.StatusOK {
		t.Fatalf("dmca failed: status=%d body=%s", dmcaRR.Code, dmcaRR.Body.String())
	}

	filingReq := httptest.NewRequest(http.MethodPost, "/api/v1/legal/regulatory-filings/generate-1099", strings.NewReader(`{"user_id":"user-9","tax_year":2025}`))
	filingReq.Header.Set("Authorization", "Bearer legal-1")
	filingReq.Header.Set("X-Actor-Role", "legal")
	filingReq.Header.Set("Idempotency-Key", "idem-http-filing")
	filingRR := httptest.NewRecorder()
	router.ServeHTTP(filingRR, filingReq)
	if filingRR.Code != http.StatusOK {
		t.Fatalf("filing failed: status=%d body=%s", filingRR.Code, filingRR.Body.String())
	}
	var filingOut contracts.SuccessResponse
	if err := json.Unmarshal(filingRR.Body.Bytes(), &filingOut); err != nil {
		t.Fatalf("decode filing response: %v", err)
	}
	filingData, _ := json.Marshal(filingOut.Data)
	var filing contracts.FilingResponse
	if err := json.Unmarshal(filingData, &filing); err != nil {
		t.Fatalf("decode filing: %v", err)
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/legal/regulatory-filings/"+filing.FilingID+"/status", nil)
	statusReq.Header.Set("Authorization", "Bearer support-1")
	statusReq.Header.Set("X-Actor-Role", "support")
	statusRR := httptest.NewRecorder()
	router.ServeHTTP(statusRR, statusReq)
	if statusRR.Code != http.StatusOK {
		t.Fatalf("filing status failed: status=%d body=%s", statusRR.Code, statusRR.Body.String())
	}
}
