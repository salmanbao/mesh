package http

import "net/http"

func NewRouter(handler *Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeSuccess(w, http.StatusOK, "ok", nil)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeSuccess(w, http.StatusOK, "ready", nil)
	})
	mux.Handle("/api/v1/developers/register", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.registerDeveloper(w, r)
	})))
	createAPIKey := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.createAPIKey(w, r)
	}))
	mux.Handle("/api/v1/developers/api-keys", createAPIKey)
	mux.Handle("/api-keys", createAPIKey)
	mux.Handle("/api/v1/developers/api-keys/{id}/rotate", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.rotateAPIKey(w, r)
	})))
	mux.Handle("/api/v1/developers/api-keys/{id}/revoke", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.revokeAPIKey(w, r)
	})))
	createWebhook := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.createWebhook(w, r)
	}))
	mux.Handle("/api/v1/developers/webhooks", createWebhook)
	mux.Handle("/webhooks", createWebhook)
	mux.Handle("/api/v1/developers/webhooks/{id}/test", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.testWebhook(w, r)
	})))
	return requestIDMiddleware(mux)
}
