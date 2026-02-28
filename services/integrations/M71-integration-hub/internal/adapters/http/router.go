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

	authorize := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.authorizeIntegration(w, r)
	}))
	mux.Handle("/api/v1/integrations/{type}/authorize", authorize)
	mux.Handle("/integrations/{type}/authorize", authorize)

	createWebhook := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.createWebhook(w, r)
	}))
	mux.Handle("/api/v1/webhooks", createWebhook)
	mux.Handle("/webhooks", createWebhook)

	createWorkflow := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.createWorkflow(w, r)
	}))
	mux.Handle("/api/v1/workflows", createWorkflow)
	mux.Handle("/workflows", createWorkflow)

	mux.Handle("/api/v1/workflows/{id}/publish", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.publishWorkflow(w, r)
	})))
	mux.Handle("/api/v1/workflows/{id}/test", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.testWorkflow(w, r)
	})))
	mux.Handle("/api/v1/webhooks/{id}/test", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.testWebhook(w, r)
	})))
	mux.Handle("/chat.postMessage", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.chatPostMessage(w, r)
	})))

	return requestIDMiddleware(mux)
}
