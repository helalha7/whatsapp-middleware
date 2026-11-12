package listener

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/helalha7/whatsapp-middleware/internal/event"
	"github.com/helalha7/whatsapp-middleware/internal/httpx"
)

type inboundMessageRegistry interface {
	RegisterInboundMessage(ctx context.Context, wabaID string, messageID string) (bool, error)
}

type inboundPublisher interface {
	Publish(ctx context.Context, message event.InboundMessage) error
}

type handler struct {
	verifyToken      string
	appSecret        string
	messageRegistry  inboundMessageRegistry
	inboundPublisher inboundPublisher
}

func RegisterRoutes(
	mux *http.ServeMux,
	verifyToken string,
	appSecret string,
	messageRegistry inboundMessageRegistry,
	inboundPublisher inboundPublisher,
) {
	h := &handler{
		verifyToken:      verifyToken,
		appSecret:        appSecret,
		messageRegistry:  messageRegistry,
		inboundPublisher: inboundPublisher,
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

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	for _, event := range events {
		isNew, err := h.messageRegistry.RegisterInboundMessage(
			ctx,
			event.WABAID,
			event.ID,
		)
		if err != nil {
			httpx.ServerErrorResponse(w, r, err)
			return
		}
		if !isNew {
			continue
		}

		err = h.inboundPublisher.Publish(ctx, event)
		if err != nil {
			httpx.ServerErrorResponse(w, r, err)
			return
		}
	}

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
