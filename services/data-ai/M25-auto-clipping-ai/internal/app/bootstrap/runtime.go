package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type Runtime struct {
	config Config
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

func NewRuntime(_ context.Context, configPath string) (Runtime, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return Runtime{}, err
	}
	if err := validateConfig(cfg); err != nil {
		return Runtime{}, err
	}
	return Runtime{config: cfg}, nil
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

func (r Runtime) router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": r.config.ServiceID,
			"mode":    "out_of_mvp",
		})
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

func (r Runtime) String() string {
	return fmt.Sprintf("%s@:%d", r.config.ServiceID, r.config.HTTPPort)
}
