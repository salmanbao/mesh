package http

import (
	"context"
	"net/http"
)

func (h *Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeMessage(w, http.StatusOK, "ok")
}

func (h *Handler) readyz(w http.ResponseWriter, _ *http.Request) {
	writeMessage(w, http.StatusOK, "ready")
}

func (h *Handler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := bearerTokenFromHeader(r.Header.Get("Authorization"))
		if err != nil {
			writeMissingBearerError(r.Context(), w, "auth_middleware")
			return
		}

		claims, err := h.service.ValidateToken(r.Context(), raw)
		if err != nil {
			writeMappedError(r.Context(), w, "auth_middleware", err)
			return
		}

		ctx := r.Context()
		ctx = contextWithToken(ctx, raw, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func contextWithToken(ctx context.Context, token string, claims any) context.Context {
	ctx = context.WithValue(ctx, ctxKeyTokenRaw, token)
	ctx = context.WithValue(ctx, ctxKeyClaims, claims)
	return ctx
}

func tokenFromContext(r *http.Request) (string, bool) {
	v := r.Context().Value(ctxKeyTokenRaw)
	token, ok := v.(string)
	return token, ok
}
