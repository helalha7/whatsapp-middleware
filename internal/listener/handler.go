package listener

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/helalha7/whatsapp-middleware/internal/sender"
)

type handler struct {
	verifyToken string
	appSecret   string
}

func Register(
	mux *http.ServeMux,
	verifyToken string,
	appSecret string,
) {
	h := &handler{
		verifyToken: verifyToken,
		appSecret:   appSecret,
	}

	mux.HandleFunc("GET /v1/webhooks", h.handleVerification)
	mux.HandleFunc("POST /v1/webhooks", h.handleNotification)
}

func (h *handler) handleVerification(
	w http.ResponseWriter,
	r *http.Request,
) {
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	if mode != "subscribe" || token != h.verifyToken {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(challenge))
}

func (h *handler) handleNotification(
	w http.ResponseWriter,
	r *http.Request,
) {
	body, err := io.ReadAll(
		http.MaxBytesReader(w, r.Body, 1<<20),
	)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if !verifySignature(
		body,
		[]byte(h.appSecret),
		r.Header.Get("X-Hub-Signature-256"),
	) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var payload payload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	events, err := normalizePayload(payload)
	if err != nil {
		http.Error(w, "failed to normalize payload", http.StatusBadRequest)
		return
	}

	for _, event := range events {
		fmt.Printf("%+v\n", event)
	}

	w.WriteHeader(http.StatusOK)

	go func() {
		if events != nil {
			s := sender.New(http.DefaultClient)
			s.SendTextMessage(
				events[0].PhoneNumberID,
				os.Getenv("access_token"),
				events[0].From,
				events[0].Text.Body,
			)
		}
	}()
}

func verifySignature(
	body []byte,
	secret []byte,
	signature string,
) bool {
	signature, found := strings.CutPrefix(signature, "sha256=")
	if !found {
		return false
	}

	received, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write(body)

	expected := mac.Sum(nil)

	return hmac.Equal(expected, received)
}
