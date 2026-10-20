package listener

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type listener struct {
	verifyToken string
}

func NewListener(verifyToken string) listener {
	return listener{
		verifyToken: verifyToken,
	}
}

func (listener listener) HandleVerification(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("hub.mode")
	challenge := r.URL.Query().Get("hub.challenge")
	verifyToken := r.URL.Query().Get("verify_token")

	if mode == "subscirbe" && verifyToken == listener.verifyToken {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(challenge))
	} else {
		w.WriteHeader(http.StatusUnauthorized)
	}
}

func (listener listener) HandleNotification(w http.ResponseWriter, r *http.Request) {
	var payload Payload
	json.NewDecoder(r.Body).Decode(&payload)

	text := payload.Entry[0].Changes[0].Value.Messages[0].Text
	fmt.Println(text)
}
