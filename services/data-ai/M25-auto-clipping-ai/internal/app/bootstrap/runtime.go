package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Runtime struct {
	config Config

	idempotencyStore     *deployIdempotencyStore
	idempotencyStoreErr  error
	idempotencyStoreInit sync.Once
}

type canonicalErrorEnvelope struct {
	Status    string             `json:"status"`
	Error     canonicalErrorBody `json:"error"`
	Timestamp string             `json:"timestamp"`
}

type canonicalErrorBody struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

type canonicalSuccessEnvelope struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

func NewRuntime(_ context.Context, configPath string) (Runtime, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return Runtime{}, err
	}
	if err := validateConfig(cfg); err != nil {
		return Runtime{}, err
	}
	store, err := newDeployIdempotencyStore(cfg.IdempotencyStorePath, deployModelIdempotencyTTL)
	if err != nil {
		return Runtime{}, fmt.Errorf("initialize deploy idempotency store: %w", err)
	}
	return Runtime{config: cfg, idempotencyStore: store}, nil
}

func (r Runtime) RunAPI(ctx context.Context) error {
	server := &http.Server{
		Addr:              ":" + strconv.Itoa(r.config.HTTPPort),
		Handler:           r.router(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err != nil {
			return err
		}
		return nil
	}
}

func (r Runtime) RunWorker(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func (r *Runtime) router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": r.config.ServiceID,
			"mode":    "out_of_mvp",
		})
	})
	mux.HandleFunc("POST /v1/admin/models/deploy", func(w http.ResponseWriter, req *http.Request) {
		if !isAdminRequest(req) {
			writeCanonicalError(
				w,
				http.StatusForbidden,
				"FORBIDDEN",
				"admin scope required",
				map[string]interface{}{"required_role": "admin"},
			)
			return
		}

		idempotencyKey := strings.TrimSpace(req.Header.Get("Idempotency-Key"))
		if idempotencyKey == "" {
			writeCanonicalError(
				w,
				http.StatusBadRequest,
				"IDEMPOTENCY_KEY_REQUIRED",
				"Idempotency-Key header is required",
				nil,
			)
			return
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			writeCanonicalError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body", nil)
			return
		}

		var payload deployModelRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			writeCanonicalError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", nil)
			return
		}
		payload.ModelName = strings.TrimSpace(payload.ModelName)
		payload.VersionTag = strings.TrimSpace(payload.VersionTag)
		payload.ModelArtifactKey = strings.TrimSpace(payload.ModelArtifactKey)
		payload.Description = strings.TrimSpace(payload.Description)
		payload.Reason = strings.TrimSpace(payload.Reason)
		if payload.ModelName == "" || payload.VersionTag == "" || payload.ModelArtifactKey == "" || payload.Reason == "" {
			writeCanonicalError(w, http.StatusBadRequest, "INVALID_REQUEST", "model_name, version_tag, model_artifact_key, and reason are required", nil)
			return
		}
		if payload.CanaryPercentage < 0 || payload.CanaryPercentage > 100 {
			writeCanonicalError(w, http.StatusBadRequest, "INVALID_REQUEST", "canary_percentage must be between 0 and 100", nil)
			return
		}

		store, err := r.ensureDeployIdempotencyStore()
		if err != nil {
			writeCanonicalError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "idempotency persistence unavailable", nil)
			return
		}

		requestHash := fmt.Sprintf("%x", sha256.Sum256(body))
		replay, replayed, err := store.replayFor(idempotencyKey, requestHash, time.Now().UTC())
		if err != nil {
			if errors.Is(err, errIdempotencyCollision) {
				writeCanonicalError(w, http.StatusConflict, "IDEMPOTENCY_COLLISION", "idempotency key reused with different payload", nil)
				return
			}
			writeCanonicalError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "idempotency persistence unavailable", nil)
			return
		}
		if replayed {
			writeCanonicalSuccess(w, http.StatusCreated, replay)
			return
		}

		deployedAt := time.Now().UTC()
		response := deployModelResponse{
			ModelVersionID:   fmt.Sprintf("m25_model_%d", deployedAt.UnixNano()),
			DeploymentStatus: fmt.Sprintf("canary_%dpct", payload.CanaryPercentage),
			DeployedAt:       deployedAt.Format(time.RFC3339),
			Message:          "model deployment accepted",
		}
		if err := store.store(idempotencyKey, requestHash, response, deployedAt); err != nil {
			if errors.Is(err, errIdempotencyCollision) {
				writeCanonicalError(w, http.StatusConflict, "IDEMPOTENCY_COLLISION", "idempotency key reused with different payload", nil)
				return
			}
			writeCanonicalError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "idempotency persistence unavailable", nil)
			return
		}
		writeCanonicalSuccess(w, http.StatusCreated, response)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGone)
		_ = json.NewEncoder(w).Encode(canonicalErrorEnvelope{
			Status: "error",
			Error: canonicalErrorBody{
				Code:    "SERVICE_OUT_OF_MVP",
				Message: "M25-Auto-Clipping-AI business APIs are disabled in MVP scope",
				Details: map[string]interface{}{
					"dependency_owner_api": r.config.ClippingToolOwnerAPIURL,
				},
			},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	})
	return mux
}

type deployModelRequest struct {
	ModelName        string `json:"model_name"`
	VersionTag       string `json:"version_tag"`
	ModelArtifactKey string `json:"model_artifact_key"`
	CanaryPercentage int    `json:"canary_percentage"`
	Description      string `json:"description"`
	Reason           string `json:"reason"`
}

type deployModelResponse struct {
	ModelVersionID   string `json:"model_version_id"`
	DeploymentStatus string `json:"deployment_status"`
	DeployedAt       string `json:"deployed_at"`
	Message          string `json:"message"`
}

func isAdminRequest(req *http.Request) bool {
	authHeader := strings.TrimSpace(req.Header.Get("Authorization"))
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(req.Header.Get("X-Actor-Role")), "admin")
}

func writeCanonicalSuccess(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(canonicalSuccessEnvelope{
		Status: "success",
		Data:   data,
	})
}

func writeCanonicalError(w http.ResponseWriter, status int, code string, message string, details map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(canonicalErrorEnvelope{
		Status: "error",
		Error: canonicalErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (r *Runtime) ensureDeployIdempotencyStore() (*deployIdempotencyStore, error) {
	r.idempotencyStoreInit.Do(func() {
		if r.idempotencyStore != nil {
			return
		}
		r.idempotencyStore, r.idempotencyStoreErr = newDeployIdempotencyStore(
			r.config.IdempotencyStorePath,
			deployModelIdempotencyTTL,
		)
	})
	if r.idempotencyStoreErr != nil {
		return nil, r.idempotencyStoreErr
	}
	return r.idempotencyStore, nil
}

func (r Runtime) String() string {
	return fmt.Sprintf("%s@:%d", r.config.ServiceID, r.config.HTTPPort)
}
