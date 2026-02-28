package http

import "net/http"

func NewRouter(handler *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.health(w, r)
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.health(w, r)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.health(w, r)
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.metrics(w, r)
	})

	mux.Handle("/configs", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handler.listConfigs(w, r)
		case http.MethodPost:
			handler.createConfig(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
		}
	})))

	mux.Handle("/purge", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.purge(w, r)
	})))

	return requestIDMiddleware(mux)
}
