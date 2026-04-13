package whatsapp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const token = "Helal212"

func VerifyWebhook(w http.ResponseWriter, r *http.Request) {
	fmt.Print(1)
	query := r.URL.Query()
	challenge := query.Get("hub.challenge")
	tok := query.Get("hub.verify_token")
	mode := query.Get("hub.mode")

	if tok == token && mode == "subscribe" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(challenge))
	}
}

func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	payload := &WebhookPayload{}
	body, _ := io.ReadAll(r.Body)
	json.Unmarshal(body, payload)
	fmt.Println(payload)
}
