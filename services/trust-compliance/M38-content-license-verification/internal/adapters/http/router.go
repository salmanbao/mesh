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

	mux.Handle("/api/v1/license/scan", authMiddleware(http.HandlerFunc(handler.scanLicense)))
	mux.Handle("/api/v1/license/appeal", authMiddleware(http.HandlerFunc(handler.fileAppeal)))
	mux.Handle("/api/v1/admin/dmca-takedown", authMiddleware(http.HandlerFunc(handler.receiveDMCA)))

	return requestIDMiddleware(mux)
}
