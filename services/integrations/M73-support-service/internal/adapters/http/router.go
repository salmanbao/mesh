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

	mux.Handle("/api/v1/support/tickets", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.createTicket(w, r)
	})))

	mux.Handle("/api/v1/support/tickets/search", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.searchTickets(w, r)
	})))

	mux.Handle("/api/v1/support/tickets/{id}", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handler.getTicket(w, r)
		case http.MethodPatch:
			handler.updateTicket(w, r)
		case http.MethodDelete:
			handler.deleteTicket(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
		}
	})))

	mux.Handle("/api/v1/support/tickets/{id}/replies", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.addReply(w, r)
	})))

	mux.Handle("/api/v1/support/tickets/{id}/csat", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.submitCSAT(w, r)
	})))

	mux.Handle("/api/v1/support/admin/tickets/{id}/assign", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.assignTicket(w, r)
	})))

	mux.Handle("/api/internal/tickets/create-from-email", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
			return
		}
		handler.createTicketFromEmail(w, r)
	})))

	return requestIDMiddleware(mux)
}
