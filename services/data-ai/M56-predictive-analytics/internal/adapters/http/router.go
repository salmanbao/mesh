package http

import "net/http"

func NewRouter(handler *Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeSuccess(w, http.StatusOK, "ok", nil)
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeSuccess(w, http.StatusOK, "ready", nil)
	})

	protected := http.NewServeMux()
	protected.HandleFunc("GET /api/v1/predictions/view-forecast", handler.getViewForecast)
	protected.HandleFunc("GET /api/v1/predictions/clip-recommendations", handler.getClipRecommendations)
	protected.HandleFunc("GET /api/v1/predictions/churn-risk", handler.getChurnRisk)
	protected.HandleFunc("POST /api/v1/campaigns/{campaign_id}/predict-success", handler.predictCampaignSuccess)

	mux.Handle("/", requestIDMiddleware(authMiddleware(protected)))
	return mux
}
