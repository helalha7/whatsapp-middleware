package listener

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/helalha7/whatsapp-middleware/internal/event"
)

type stubMessageRegistry struct {
	register func(context.Context, string, string) (bool, error)
}

type stubInboundPublisher struct {
	messages []event.InboundMessage
	err      error
}

func (p *stubInboundPublisher) Publish(
	_ context.Context,
	message event.InboundMessage,
) error {
	if p.err != nil {
		return p.err
	}

	p.messages = append(p.messages, message)
	return nil
}

func (s *stubMessageRegistry) RegisterInboundMessage(
	ctx context.Context,
	wabaID string,
	messageID string,
) (bool, error) {
	return s.register(ctx, wabaID, messageID)
}

func TestHandleNotificationIdempotency(t *testing.T) {
	tests := []struct {
		name       string
		isNew      bool
		storeError error
		queueError error
		wantStatus int
		wantQueued bool
	}{
		{
			name:       "queues first delivery",
			isNew:      true,
			wantStatus: http.StatusOK,
			wantQueued: true,
		},
		{
			name:       "ignores duplicate delivery",
			wantStatus: http.StatusOK,
		},
		{
			name:       "returns server error when registration fails",
			storeError: errors.New("database unavailable"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "returns server error when publishing fails",
			isNew:      true,
			queueError: errors.New("queue unavailable"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := &stubMessageRegistry{
				register: func(_ context.Context, wabaID string, messageID string) (bool, error) {
					if wabaID != "2161887421343402" {
						t.Errorf("expected WABA ID 2161887421343402, got %q", wabaID)
					}
					if messageID != "wamid.message-id" {
						t.Errorf("expected message ID wamid.message-id, got %q", messageID)
					}
					return tt.isNew, tt.storeError
				},
			}
			publisher := &stubInboundPublisher{err: tt.queueError}
			h := &handler{
				appSecret:        "test-secret",
				messageRegistry:  registry,
				inboundPublisher: publisher,
			}

			body := validWebhookPayload()
			request := httptest.NewRequest(http.MethodPost, "/v1/webhooks", strings.NewReader(body))
			request.Header.Set("X-Hub-Signature-256", signPayload(body, "test-secret"))
			recorder := httptest.NewRecorder()

			h.handleNotification(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, recorder.Code)
			}
			if queued := len(publisher.messages) == 1; queued != tt.wantQueued {
				t.Errorf("expected queued=%t, got %t", tt.wantQueued, queued)
			}
		})
	}
}

func signPayload(payload string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func validWebhookPayload() string {
	return `{
		"object":"whatsapp_business_account",
		"entry":[{
			"id":"2161887421343402",
			"changes":[{
				"field":"messages",
				"value":{
					"messaging_product":"whatsapp",
					"metadata":{
						"display_phone_number":"15552050459",
						"phone_number_id":"1257434784125832"
					},
					"contacts":[{
						"profile":{"name":"Customer"},
						"wa_id":"15551234567"
					}],
					"messages":[{
						"from":"15551234567",
						"id":"wamid.message-id",
						"timestamp":"1794312000",
						"type":"text",
						"text":{"body":"Hello"}
					}]
				}
			}]
		}]
	}`
}
