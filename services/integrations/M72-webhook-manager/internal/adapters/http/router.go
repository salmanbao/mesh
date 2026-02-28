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

	mux.Handle("/api/v1/webhooks", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.createWebhook(w, r)
	})))

	mux.Handle("/api/v1/webhooks/{id}", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handler.getWebhook(w, r)
		case http.MethodPatch:
			handler.updateWebhook(w, r)
		case http.MethodDelete:
			handler.deleteWebhook(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
		}
	})))

	mux.Handle("/api/v1/webhooks/{id}/test", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.testWebhook(w, r)
	})))

	mux.Handle("/api/v1/webhooks/{id}/deliveries", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.listDeliveries(w, r)
	})))

	mux.Handle("/api/v1/webhooks/{id}/analytics", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.analytics(w, r)
	})))

	mux.Handle("/api/v1/webhooks/{id}/enable", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.enableWebhook(w, r)
	})))

	mux.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.receiveCompatibilityWebhook(w, r)
	})

	return requestIDMiddleware(mux)
}
