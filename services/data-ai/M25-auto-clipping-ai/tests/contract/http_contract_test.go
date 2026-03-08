package contract

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M25-auto-clipping-ai/internal/app/bootstrap"
)

func TestAdminDeployRouteMatchesPublishedContract(t *testing.T) {
	port := reserveFreePort(t)
	t.Setenv("HTTP_PORT", strconv.Itoa(port))
	t.Setenv("M25_RUNTIME_MODE", "test")
	t.Setenv("M25_IDEMPOTENCY_STORE_PATH", filepath.Join(t.TempDir(), "idempotency.json"))

	rt, err := bootstrap.NewRuntime(context.Background(), "testdata/does-not-exist.yaml")
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- rt.RunAPI(ctx)
	}()
	waitForHealth(t, port)

	reqBody := `{"model_name":"xgboost_ensemble","version_tag":"v1.2.3","model_artifact_key":"s3://models/xgboost/v1.2.3.bin","canary_percentage":5,"reason":"weekly rollout"}`
	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:"+strconv.Itoa(port)+"/v1/admin/models/deploy", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer admin-1")
	req.Header.Set("X-Actor-Role", "admin")
	req.Header.Set("Idempotency-Key", "idem-contract-m25-deploy")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("deploy request: %v", err)
	}
	defer resp.Body.Close()
	rawBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("unexpected deploy status: got=%d body=%s", resp.StatusCode, string(rawBody))
	}
	if !strings.Contains(string(rawBody), `"status":"success"`) || !strings.Contains(string(rawBody), `"model_version_id"`) {
		t.Fatalf("unexpected deploy response body=%s", string(rawBody))
	}

	specPath := filepath.Join("..", "..", "..", "..", "..", "contracts", "openapi", "m25-auto-clipping-ai.yaml")
	specRaw, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read openapi spec: %v", err)
	}
	spec := string(specRaw)
	for _, required := range []string{
		"/v1/admin/models/deploy:",
		"post:",
		"Idempotency-Key",
		"X-Actor-Role",
	} {
		if !strings.Contains(spec, required) {
			t.Fatalf("openapi contract missing %q", required)
		}
	}

	cancel()
	select {
	case runErr := <-errCh:
		if runErr != nil {
			t.Fatalf("runtime shutdown error: %v", runErr)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("runtime did not shut down after context cancellation")
	}
}

func reserveFreePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func waitForHealth(t *testing.T, port int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	healthURL := "http://127.0.0.1:" + strconv.Itoa(port) + "/health"
	for time.Now().Before(deadline) {
		resp, err := http.Get(healthURL)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("health endpoint did not become ready: %s", healthURL)
}
