package listener

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/helalha7/whatsapp-middleware/internal/event"
	"github.com/helalha7/whatsapp-middleware/internal/httpx"
)

type inboundMessageRegistry interface {
	RegisterInboundMessage(ctx context.Context, wabaID string, messageID string) (bool, error)
	ReleaseInboundMessage(ctx context.Context, wabaID string, messageID string) error
}

type inboundPublisher interface {
	Publish(ctx context.Context, message event.InboundMessage) error
}

const (
	maxWebhookBodyBytes = 1 << 20
	webhookWorkTimeout  = 5 * time.Second
)

var (
	errInvalidNotificationJSON    = errors.New("invalid notification JSON")
	errInvalidNotificationPayload = errors.New("invalid notification payload")
	errNormalizeNotification      = errors.New("normalize notification")
)

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
		http.MaxBytesReader(w, r.Body, maxWebhookBodyBytes),
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

	events, err := decodeNotification(body)
	if err != nil {
		switch {
		case errors.Is(err, errInvalidNotificationJSON):
			http.Error(w, "invalid JSON", http.StatusBadRequest)
		case errors.Is(err, errNormalizeNotification):
			http.Error(w, "failed to normalize payload", http.StatusBadRequest)
		default:
			http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		}
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), webhookWorkTimeout)
	defer cancel()

	if err := h.publishEvents(ctx, r.Context(), events); err != nil {
		httpx.ServerErrorResponse(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func decodeNotification(body []byte) ([]event.InboundMessage, error) {
	var payload payload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("%w: %v", errInvalidNotificationJSON, err)
	}
	if err := validatePayload(payload); err != nil {
		return nil, fmt.Errorf("%w: %v", errInvalidNotificationPayload, err)
	}

	events, err := normalizePayload(payload)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errNormalizeNotification, err)
	}
	return events, nil
}

func (h *handler) publishEvents(
	ctx context.Context,
	releaseBaseCtx context.Context,
	events []event.InboundMessage,
) error {
	for _, event := range events {
		isNew, err := h.messageRegistry.RegisterInboundMessage(
			ctx,
			event.BusinessAccountID,
			event.WhatsAppMessageID,
		)
		if err != nil {
			return err
		}
		if !isNew {
			continue
		}

		if err := h.inboundPublisher.Publish(ctx, event); err != nil {
			releaseCtx, releaseCancel := context.WithTimeout(
				context.WithoutCancel(releaseBaseCtx),
				webhookWorkTimeout,
			)
			releaseErr := h.messageRegistry.ReleaseInboundMessage(
				releaseCtx,
				event.BusinessAccountID,
				event.WhatsAppMessageID,
			)
			releaseCancel()

			return errors.Join(err, releaseErr)
		}
	}

	return nil
}

func validatePayload(
	p payload,
) error {
	if p.Object != "whatsapp_business_account" {
		return fmt.Errorf("unexpected webhook object %q", p.Object)
	}
	if len(p.Entry) == 0 {
		return errors.New("webhook payload contains no entries")
	}

	return nil
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
