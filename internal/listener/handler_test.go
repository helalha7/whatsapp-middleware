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
	release  func(context.Context, string, string) error
}

type stubInboundPublisher struct {
	messages []event.InboundMessage
	err      error
	publish  func(context.Context, event.InboundMessage) error
}

func (p *stubInboundPublisher) Publish(
	ctx context.Context,
	message event.InboundMessage,
) error {
	if p.publish != nil {
		return p.publish(ctx, message)
	}
	if p.err != nil {
		return p.err
	}

	p.messages = append(p.messages, message)
	return nil
}

func TestHandleNotificationProcessesBatch(t *testing.T) {
	body := strings.Replace(
		validWebhookPayload(),
		`"messages":[{`,
		`"messages":[{
						"from":"15551234567",
						"id":"wamid.first",
						"timestamp":"1794311999",
						"type":"text",
						"text":{"body":"First"}
					},{`,
		1,
	)

	registered := make([]string, 0, 2)
	registry := &stubMessageRegistry{
		register: func(_ context.Context, _ string, messageID string) (bool, error) {
			registered = append(registered, messageID)
			return true, nil
		},
	}
	publisher := &stubInboundPublisher{}
	h := &handler{
		appSecret:        "test-secret",
		messageRegistry:  registry,
		inboundPublisher: publisher,
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/webhooks", strings.NewReader(body))
	request.Header.Set("X-Hub-Signature-256", signPayload(body, "test-secret"))
	recorder := httptest.NewRecorder()

	h.handleNotification(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if len(registered) != 2 || len(publisher.messages) != 2 {
		t.Fatalf("expected 2 registered and published messages, got registered=%d published=%d", len(registered), len(publisher.messages))
	}
	if publisher.messages[0].WhatsAppMessageID != "wamid.first" || publisher.messages[1].WhatsAppMessageID != "wamid.message-id" {
		t.Errorf("unexpected published messages: %+v", publisher.messages)
	}
}

func TestPublishEventsHandlesMixedBatch(t *testing.T) {
	events := []event.InboundMessage{
		{WhatsAppMessageID: "wamid.duplicate", BusinessAccountID: "waba-id"},
		{WhatsAppMessageID: "wamid.new", BusinessAccountID: "waba-id"},
	}
	registry := &stubMessageRegistry{
		register: func(_ context.Context, _ string, messageID string) (bool, error) {
			return messageID == "wamid.new", nil
		},
	}
	publisher := &stubInboundPublisher{}
	h := &handler{messageRegistry: registry, inboundPublisher: publisher}

	err := h.publishEvents(context.Background(), context.Background(), events)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(publisher.messages) != 1 || publisher.messages[0].WhatsAppMessageID != "wamid.new" {
		t.Fatalf("expected only new message to be published, got %+v", publisher.messages)
	}
}

func TestPublishEventsReleasesFailedMessageInBatch(t *testing.T) {
	queueError := errors.New("queue unavailable")
	events := []event.InboundMessage{
		{WhatsAppMessageID: "wamid.first", BusinessAccountID: "waba-id"},
		{WhatsAppMessageID: "wamid.second", BusinessAccountID: "waba-id"},
	}
	releasedID := ""
	registry := &stubMessageRegistry{
		register: func(context.Context, string, string) (bool, error) {
			return true, nil
		},
		release: func(_ context.Context, _ string, messageID string) error {
			releasedID = messageID
			return nil
		},
	}
	published := make([]string, 0, 1)
	publisher := &stubInboundPublisher{
		publish: func(_ context.Context, message event.InboundMessage) error {
			if message.WhatsAppMessageID == "wamid.second" {
				return queueError
			}
			published = append(published, message.WhatsAppMessageID)
			return nil
		},
	}
	h := &handler{messageRegistry: registry, inboundPublisher: publisher}

	err := h.publishEvents(context.Background(), context.Background(), events)
	if !errors.Is(err, queueError) {
		t.Fatalf("expected error %v, got %v", queueError, err)
	}
	if len(published) != 1 || published[0] != "wamid.first" {
		t.Errorf("unexpected published messages: %v", published)
	}
	if releasedID != "wamid.second" {
		t.Errorf("expected failed message to be released, got %q", releasedID)
	}
}

func (s *stubMessageRegistry) RegisterInboundMessage(
	ctx context.Context,
	wabaID string,
	messageID string,
) (bool, error) {
	return s.register(ctx, wabaID, messageID)
}

func (s *stubMessageRegistry) ReleaseInboundMessage(
	ctx context.Context,
	wabaID string,
	messageID string,
) error {
	if s.release == nil {
		return nil
	}

	return s.release(ctx, wabaID, messageID)
}

func TestHandleNotificationIdempotency(t *testing.T) {
	tests := []struct {
		name         string
		isNew        bool
		storeError   error
		queueError   error
		releaseError error
		wantStatus   int
		wantQueued   bool
		wantRelease  bool
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
			name:        "returns server error when publishing fails",
			isNew:       true,
			queueError:  errors.New("queue unavailable"),
			wantStatus:  http.StatusInternalServerError,
			wantRelease: true,
		},
		{
			name:         "returns server error when releasing registration fails",
			isNew:        true,
			queueError:   errors.New("queue unavailable"),
			releaseError: errors.New("database unavailable"),
			wantStatus:   http.StatusInternalServerError,
			wantRelease:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			released := false
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
				release: func(_ context.Context, wabaID string, messageID string) error {
					released = true
					if wabaID != "2161887421343402" || messageID != "wamid.message-id" {
						t.Errorf("released unexpected message %q/%q", wabaID, messageID)
					}
					return tt.releaseError
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
			if released != tt.wantRelease {
				t.Errorf("expected released=%t, got %t", tt.wantRelease, released)
			}
		})
	}
}

func TestHandleVerification(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "accepts valid verification",
			query:      "?hub.mode=subscribe&hub.verify_token=verify-me&hub.challenge=challenge-value",
			wantStatus: http.StatusOK,
			wantBody:   "challenge-value",
		},
		{
			name:       "rejects invalid token",
			query:      "?hub.mode=subscribe&hub.verify_token=wrong-token&hub.challenge=challenge-value",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "rejects invalid mode",
			query:      "?hub.mode=invalid&hub.verify_token=verify-me&hub.challenge=challenge-value",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &handler{verifyToken: "verify-me"}
			request := httptest.NewRequest(http.MethodGet, "/v1/webhooks"+tt.query, nil)
			recorder := httptest.NewRecorder()

			h.handleVerification(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, recorder.Code)
			}
			if body := recorder.Body.String(); body != tt.wantBody {
				t.Errorf("expected body %q, got %q", tt.wantBody, body)
			}
		})
	}
}

func TestHandleNotificationRejectsInvalidRequests(t *testing.T) {
	validBody := validWebhookPayload()
	tests := []struct {
		name       string
		body       string
		signature  string
		wantStatus int
	}{
		{
			name:       "invalid signature",
			body:       validBody,
			signature:  "sha256=invalid",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "malformed JSON",
			body:       `{"object":`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unexpected object",
			body:       strings.Replace(validBody, "whatsapp_business_account", "unexpected", 1),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing entries",
			body:       `{"object":"whatsapp_business_account","entry":[]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid timestamp",
			body:       strings.Replace(validBody, `"1794312000"`, `"invalid"`, 1),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unexpected messaging product",
			body:       strings.Replace(validBody, `"messaging_product":"whatsapp"`, `"messaging_product":"other"`, 1),
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := &stubMessageRegistry{
				register: func(context.Context, string, string) (bool, error) {
					t.Fatal("message registry should not be called")
					return false, nil
				},
			}
			h := &handler{
				appSecret:        "test-secret",
				messageRegistry:  registry,
				inboundPublisher: &stubInboundPublisher{},
			}
			signature := tt.signature
			if signature == "" {
				signature = signPayload(tt.body, "test-secret")
			}

			request := httptest.NewRequest(http.MethodPost, "/v1/webhooks", strings.NewReader(tt.body))
			request.Header.Set("X-Hub-Signature-256", signature)
			recorder := httptest.NewRecorder()

			h.handleNotification(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, recorder.Code)
			}
		})
	}
}

func TestHandleNotificationIgnoresUnrelatedChanges(t *testing.T) {
	body := strings.Replace(validWebhookPayload(), `"field":"messages"`, `"field":"account_update"`, 1)
	registry := &stubMessageRegistry{
		register: func(context.Context, string, string) (bool, error) {
			t.Fatal("message registry should not be called")
			return false, nil
		},
	}
	h := &handler{
		appSecret:        "test-secret",
		messageRegistry:  registry,
		inboundPublisher: &stubInboundPublisher{},
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/webhooks", strings.NewReader(body))
	request.Header.Set("X-Hub-Signature-256", signPayload(body, "test-secret"))
	recorder := httptest.NewRecorder()

	h.handleNotification(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
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
