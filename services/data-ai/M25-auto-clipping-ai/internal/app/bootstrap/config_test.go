package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestRuntime(t *testing.T) Runtime {
	t.Helper()
	return Runtime{
		config: Config{
			ServiceID:               "M25-Auto-Clipping-AI",
			HTTPPort:                8086,
			ClippingToolOwnerAPIURL: "http://m24-clipping-tool-service:8080",
			IdempotencyStorePath:    filepath.Join(t.TempDir(), "idempotency.json"),
		},
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	cfg, err := LoadConfig("testdata/does-not-exist.yaml")
	if err != nil {
		t.Fatalf("expected defaults, got err=%v", err)
	}
	if cfg.ServiceID != "M25-Auto-Clipping-AI" {
		t.Fatalf("unexpected service id: %s", cfg.ServiceID)
	}
	if cfg.HTTPPort != 8086 {
		t.Fatalf("expected default http port 8086, got %d", cfg.HTTPPort)
	}
	if cfg.ClippingToolOwnerAPIURL == "" {
		t.Fatal("expected default clipping tool owner API URL to be set")
	}
	if cfg.IdempotencyStorePath == "" {
		t.Fatal("expected default idempotency store path to be set")
	}
}

func TestValidateConfigRequiresOwnerAPIURL(t *testing.T) {
	err := validateConfig(Config{
		ServiceID:               "M25-Auto-Clipping-AI",
		HTTPPort:                8086,
		ClippingToolOwnerAPIURL: "",
		IdempotencyStorePath:    filepath.Join(t.TempDir(), "idempotency.json"),
	})
	if err == nil {
		t.Fatal("expected validation failure when owner API URL is missing")
	}
}

func TestRouterHealthAndOutOfMVPEndpoints(t *testing.T) {
	runtime := newTestRuntime(t)
	router := runtime.router()

	healthReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthRR := httptest.NewRecorder()
	router.ServeHTTP(healthRR, healthReq)
	if healthRR.Code != http.StatusOK {
		t.Fatalf("expected 200 for /health, got %d", healthRR.Code)
	}

	outReq := httptest.NewRequest(http.MethodGet, "/v1/suggestions/abc", nil)
	outRR := httptest.NewRecorder()
	router.ServeHTTP(outRR, outReq)
	if outRR.Code != http.StatusGone {
		t.Fatalf("expected 410 for out-of-MVP API, got %d", outRR.Code)
	}
}

func TestValidateConfigProductionModeRequiresExplicitOwnerURL(t *testing.T) {
	t.Setenv("M25_RUNTIME_MODE", "production")
	t.Setenv("M24_CLIPPING_TOOL_OWNER_API_URL", "")
	t.Setenv("M25_IDEMPOTENCY_STORE_PATH", filepath.Join(t.TempDir(), "idempotency.json"))

	err := validateConfig(Config{
		ServiceID:               "M25-Auto-Clipping-AI",
		HTTPPort:                8086,
		ClippingToolOwnerAPIURL: "http://m24-clipping-tool-service:8080",
		IdempotencyStorePath:    filepath.Join(t.TempDir(), "idempotency.json"),
	})
	if err == nil || !strings.Contains(err.Error(), "M24_CLIPPING_TOOL_OWNER_API_URL is required in production runtime") {
		t.Fatalf("expected production env validation error, got err=%v", err)
	}
}

func TestValidateConfigProductionModeRequiresAbsoluteOwnerURL(t *testing.T) {
	t.Setenv("M25_RUNTIME_MODE", "production")
	t.Setenv("M24_CLIPPING_TOOL_OWNER_API_URL", "/relative-owner")
	t.Setenv("M25_IDEMPOTENCY_STORE_PATH", filepath.Join(t.TempDir(), "idempotency.json"))

	err := validateConfig(Config{
		ServiceID:               "M25-Auto-Clipping-AI",
		HTTPPort:                8086,
		ClippingToolOwnerAPIURL: "/relative-owner",
		IdempotencyStorePath:    filepath.Join(t.TempDir(), "idempotency.json"),
	})
	if err == nil || !strings.Contains(err.Error(), "must be an absolute URL") {
		t.Fatalf("expected absolute URL validation error, got err=%v", err)
	}
}

func TestValidateConfigProductionModeAcceptsExplicitAbsoluteOwnerURL(t *testing.T) {
	t.Setenv("M25_RUNTIME_MODE", "production")
	t.Setenv("M24_CLIPPING_TOOL_OWNER_API_URL", "https://m24.internal")
	t.Setenv("M25_IDEMPOTENCY_STORE_PATH", filepath.Join(t.TempDir(), "idempotency.json"))

	err := validateConfig(Config{
		ServiceID:               "M25-Auto-Clipping-AI",
		HTTPPort:                8086,
		ClippingToolOwnerAPIURL: "https://m24.internal",
		IdempotencyStorePath:    filepath.Join(t.TempDir(), "idempotency.json"),
	})
	if err != nil {
		t.Fatalf("expected valid production config, got err=%v", err)
	}
}

func TestLoadConfigIgnoresTemplateOwnerURLFromFile(t *testing.T) {
	dir := t.TempDir()
	configPath := dir + "/default.yaml"
	raw := []byte("service:\n  id: M25-Auto-Clipping-AI\ndependencies:\n  clipping_tool_owner_api_url: ${M24_CLIPPING_TOOL_OWNER_API_URL}\npersistence:\n  idempotency_store_path: ${M25_IDEMPOTENCY_STORE_PATH}\n")
	if err := os.WriteFile(configPath, raw, 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.ClippingToolOwnerAPIURL != "http://m24-clipping-tool-service:8080" {
		t.Fatalf("expected default owner URL fallback, got=%s", cfg.ClippingToolOwnerAPIURL)
	}
	if cfg.IdempotencyStorePath != "data/m25-admin-model-deploy-idempotency.json" {
		t.Fatalf("expected default idempotency store path fallback, got=%s", cfg.IdempotencyStorePath)
	}
}

func TestAdminDeployRequiresAdminScope(t *testing.T) {
	runtime := newTestRuntime(t)
	router := runtime.router()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/models/deploy", strings.NewReader(`{"model_name":"xgboost_ensemble","version_tag":"v1.2.3","model_artifact_key":"s3://models/xgboost/v1.2.3.bin","canary_percentage":5,"reason":"weekly rollout"}`))
	req.Header.Set("Idempotency-Key", "idem-admin-scope")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden without admin scope, got=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminDeployEnforcesIdempotency(t *testing.T) {
	runtime := newTestRuntime(t)
	router := runtime.router()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/models/deploy", strings.NewReader(`{"model_name":"xgboost_ensemble","version_tag":"v1.2.3","model_artifact_key":"s3://models/xgboost/v1.2.3.bin","canary_percentage":5,"reason":"weekly rollout"}`))
	req.Header.Set("Authorization", "Bearer admin-1")
	req.Header.Set("X-Actor-Role", "admin")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected idempotency failure, got=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminDeployIdempotentReplayAndCollision(t *testing.T) {
	runtime := newTestRuntime(t)
	router := runtime.router()
	idempotencyKey := "idem-admin-deploy"
	firstPayload := `{"model_name":"xgboost_ensemble","version_tag":"v1.2.3","model_artifact_key":"s3://models/xgboost/v1.2.3.bin","canary_percentage":5,"description":"weekly model rollout","reason":"weekly rollout"}`
	secondPayload := `{"model_name":"xgboost_ensemble","version_tag":"v1.2.4","model_artifact_key":"s3://models/xgboost/v1.2.4.bin","canary_percentage":10,"description":"new candidate","reason":"weekly rollout"}`

	makeRequest := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/admin/models/deploy", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer admin-1")
		req.Header.Set("X-Actor-Role", "admin")
		req.Header.Set("Idempotency-Key", idempotencyKey)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		return rr
	}

	first := makeRequest(firstPayload)
	if first.Code != http.StatusCreated {
		t.Fatalf("expected created for first deploy request, got=%d body=%s", first.Code, first.Body.String())
	}

	second := makeRequest(firstPayload)
	if second.Code != http.StatusCreated {
		t.Fatalf("expected created replay for matching idempotency payload, got=%d body=%s", second.Code, second.Body.String())
	}

	var firstEnv canonicalSuccessEnvelope
	if err := json.Unmarshal(first.Body.Bytes(), &firstEnv); err != nil {
		t.Fatalf("decode first response: %v", err)
	}
	var secondEnv canonicalSuccessEnvelope
	if err := json.Unmarshal(second.Body.Bytes(), &secondEnv); err != nil {
		t.Fatalf("decode second response: %v", err)
	}
	firstData, _ := json.Marshal(firstEnv.Data)
	secondData, _ := json.Marshal(secondEnv.Data)
	var firstModel deployModelResponse
	if err := json.Unmarshal(firstData, &firstModel); err != nil {
		t.Fatalf("decode first model response: %v", err)
	}
	var secondModel deployModelResponse
	if err := json.Unmarshal(secondData, &secondModel); err != nil {
		t.Fatalf("decode second model response: %v", err)
	}
	if firstModel.ModelVersionID == "" {
		t.Fatal("expected model_version_id in first response")
	}
	if firstModel.ModelVersionID != secondModel.ModelVersionID {
		t.Fatalf("expected idempotent replay to return same model_version_id: first=%s second=%s", firstModel.ModelVersionID, secondModel.ModelVersionID)
	}

	collision := makeRequest(secondPayload)
	if collision.Code != http.StatusConflict {
		t.Fatalf("expected idempotency collision on different payload, got=%d body=%s", collision.Code, collision.Body.String())
	}
}

func TestValidateConfigProductionModeRequiresExplicitIdempotencyStorePath(t *testing.T) {
	t.Setenv("M25_RUNTIME_MODE", "production")
	t.Setenv("M24_CLIPPING_TOOL_OWNER_API_URL", "https://m24.internal")
	t.Setenv("M25_IDEMPOTENCY_STORE_PATH", "")

	err := validateConfig(Config{
		ServiceID:               "M25-Auto-Clipping-AI",
		HTTPPort:                8086,
		ClippingToolOwnerAPIURL: "https://m24.internal",
		IdempotencyStorePath:    filepath.Join(t.TempDir(), "idempotency.json"),
	})
	if err == nil || !strings.Contains(err.Error(), "M25_IDEMPOTENCY_STORE_PATH is required in production runtime") {
		t.Fatalf("expected production idempotency env validation error, got err=%v", err)
	}
}

func TestValidateConfigProductionModeRequiresAbsoluteIdempotencyStorePath(t *testing.T) {
	t.Setenv("M25_RUNTIME_MODE", "production")
	t.Setenv("M24_CLIPPING_TOOL_OWNER_API_URL", "https://m24.internal")
	t.Setenv("M25_IDEMPOTENCY_STORE_PATH", "relative/idempotency.json")

	err := validateConfig(Config{
		ServiceID:               "M25-Auto-Clipping-AI",
		HTTPPort:                8086,
		ClippingToolOwnerAPIURL: "https://m24.internal",
		IdempotencyStorePath:    "relative/idempotency.json",
	})
	if err == nil || !strings.Contains(err.Error(), "must be an absolute path") {
		t.Fatalf("expected absolute idempotency path validation error, got err=%v", err)
	}
}

func TestAdminDeployReplayPersistsAcrossRuntimeRestart(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "idempotency.json")
	first := Runtime{config: Config{
		ServiceID:               "M25-Auto-Clipping-AI",
		HTTPPort:                8086,
		ClippingToolOwnerAPIURL: "http://m24-clipping-tool-service:8080",
		IdempotencyStorePath:    storePath,
	}}
	second := Runtime{config: Config{
		ServiceID:               "M25-Auto-Clipping-AI",
		HTTPPort:                8086,
		ClippingToolOwnerAPIURL: "http://m24-clipping-tool-service:8080",
		IdempotencyStorePath:    storePath,
	}}

	makeRequest := func(router http.Handler) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/admin/models/deploy", strings.NewReader(`{"model_name":"xgboost_ensemble","version_tag":"v1.2.3","model_artifact_key":"s3://models/xgboost/v1.2.3.bin","canary_percentage":5,"reason":"weekly rollout"}`))
		req.Header.Set("Authorization", "Bearer admin-1")
		req.Header.Set("X-Actor-Role", "admin")
		req.Header.Set("Idempotency-Key", "idem-restart-safe")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		return rr
	}

	firstResp := makeRequest(first.router())
	if firstResp.Code != http.StatusCreated {
		t.Fatalf("expected first deploy created, got=%d body=%s", firstResp.Code, firstResp.Body.String())
	}
	secondResp := makeRequest(second.router())
	if secondResp.Code != http.StatusCreated {
		t.Fatalf("expected replay deploy created, got=%d body=%s", secondResp.Code, secondResp.Body.String())
	}

	var firstEnv canonicalSuccessEnvelope
	if err := json.Unmarshal(firstResp.Body.Bytes(), &firstEnv); err != nil {
		t.Fatalf("decode first response: %v", err)
	}
	var secondEnv canonicalSuccessEnvelope
	if err := json.Unmarshal(secondResp.Body.Bytes(), &secondEnv); err != nil {
		t.Fatalf("decode second response: %v", err)
	}
	firstData, _ := json.Marshal(firstEnv.Data)
	secondData, _ := json.Marshal(secondEnv.Data)
	var firstModel deployModelResponse
	if err := json.Unmarshal(firstData, &firstModel); err != nil {
		t.Fatalf("decode first model response: %v", err)
	}
	var secondModel deployModelResponse
	if err := json.Unmarshal(secondData, &secondModel); err != nil {
		t.Fatalf("decode second model response: %v", err)
	}
	if firstModel.ModelVersionID == "" || secondModel.ModelVersionID == "" {
		t.Fatalf("expected non-empty model_version_id values: first=%q second=%q", firstModel.ModelVersionID, secondModel.ModelVersionID)
	}
	if firstModel.ModelVersionID != secondModel.ModelVersionID {
		t.Fatalf("expected persisted replay model_version_id to match after restart: first=%s second=%s", firstModel.ModelVersionID, secondModel.ModelVersionID)
	}
}
