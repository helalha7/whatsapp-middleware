package webhook

import "net/http"

func RegisterRoutes(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("GET /webhook", h.VerifyWebhook)
	mux.HandleFunc("POST /webhook", h.HandleWebhook)
}
