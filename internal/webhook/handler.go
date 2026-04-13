package webhook

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// token is the verification token that must match the token
// configured in the Meta WhatsApp webhook settings.
const token = "Helal212"

type Handler struct {
}

// VerifyWebhook handles the webhook verification challenge from Meta.
func (handler *Handler) VerifyWebhook(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	challenge := query.Get("hub.challenge")
	tok := query.Get("hub.verify_token")
	mode := query.Get("hub.mode")

	if tok == token && mode == "subscribe" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(challenge))
		return
	}

	http.Error(w, "forbidden", http.StatusForbidden)
}

// HandleWebhook handles incoming WhatsApp webhook events.
func (handler *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	payload := &WebhookPayload{}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(body, payload)
	if err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}

	fmt.Println(payload)
	w.WriteHeader(http.StatusOK)
}
