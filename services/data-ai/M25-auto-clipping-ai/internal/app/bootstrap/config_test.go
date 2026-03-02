package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
}

func TestValidateConfigRequiresOwnerAPIURL(t *testing.T) {
	err := validateConfig(Config{
		ServiceID:               "M25-Auto-Clipping-AI",
		HTTPPort:                8086,
		ClippingToolOwnerAPIURL: "",
	})
	if err == nil {
		t.Fatal("expected validation failure when owner API URL is missing")
	}
}

func TestRouterHealthAndOutOfMVPEndpoints(t *testing.T) {
	runtime := Runtime{config: Config{
		ServiceID:               "M25-Auto-Clipping-AI",
		HTTPPort:                8086,
		ClippingToolOwnerAPIURL: "http://m24-clipping-tool-service:8080",
	}}
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
