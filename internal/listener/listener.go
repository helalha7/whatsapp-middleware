package listener

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Listener struct {
	verifyToken string
	appSecret   string
}

func New(
	verifyToken string,
	appSecret string,
) *Listener {
	return &Listener{
		verifyToken: verifyToken,
		appSecret:   appSecret,
	}
}

func (l *Listener) HandleVerification(
	w http.ResponseWriter,
	r *http.Request,
) {
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	if mode != "subscribe" || token != l.verifyToken {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(challenge))
}

func (Listener *Listener) HandleNotification(
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
		[]byte(Listener.appSecret),
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

	// Not every webhook contains a text message.
	if len(payload.Entry) == 0 ||
		len(payload.Entry[0].Changes) == 0 ||
		len(payload.Entry[0].Changes[0].Value.Messages) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	text := payload.Entry[0].Changes[0].Value.Messages[0].Text.Body
	fmt.Println(text)

	w.WriteHeader(http.StatusOK)
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
