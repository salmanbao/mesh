package bootstrap

import "net/http"

// Build wires runtime dependencies for this service and starts the API server.
func Build() error {
	cfg := loadConfig()
	runtime := NewRuntime(cfg)
	return http.ListenAndServe(runtime.Addr, runtime.Router)
}
