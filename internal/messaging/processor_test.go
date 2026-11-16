package messaging

import (
	"context"
	"errors"
	"testing"

	"github.com/helalha7/whatsapp-middleware/internal/event"
)

type stubAccessTokenStore struct {
	token  string
	err    error
	wabaID string
}

func (s *stubAccessTokenStore) GetAccessToken(
	_ context.Context,
	wabaID string,
) (string, error) {
	s.wabaID = wabaID
	return s.token, s.err
}

type stubTextMessageSender struct {
	senderPhoneNumberID string
	senderAccessToken   string
	receiverPhoneNumber string
	message             string
	err                 error
}

func (s *stubTextMessageSender) SendTextMessage(
	_ context.Context,
	senderPhoneNumberID string,
	senderAccessToken string,
	receiverPhoneNumber string,
	message string,
) error {
	s.senderPhoneNumberID = senderPhoneNumberID
	s.senderAccessToken = senderAccessToken
	s.receiverPhoneNumber = receiverPhoneNumber
	s.message = message
	return s.err
}

func TestProcessEchoesTextMessage(t *testing.T) {
	tokens := &stubAccessTokenStore{token: "access-token"}
	sender := &stubTextMessageSender{}
	processor := NewProcessor(tokens, sender)

	err := processor.Process(context.Background(), event.InboundMessage{
		WABAID:        "waba-id",
		PhoneNumberID: "phone-number-id",
		From:          "15551234567",
		Type:          event.MessageTypeText,
		Text:          &event.TextContent{Body: "Hello"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tokens.wabaID != "waba-id" {
		t.Errorf("expected WABA ID waba-id, got %q", tokens.wabaID)
	}
	if sender.senderPhoneNumberID != "phone-number-id" {
		t.Errorf("unexpected sender phone number ID %q", sender.senderPhoneNumberID)
	}
	if sender.senderAccessToken != "access-token" {
		t.Errorf("unexpected access token %q", sender.senderAccessToken)
	}
	if sender.receiverPhoneNumber != "15551234567" || sender.message != "Hello" {
		t.Errorf("unexpected echoed message recipient=%q body=%q", sender.receiverPhoneNumber, sender.message)
	}
}

func TestProcessIgnoresUnsupportedMessageType(t *testing.T) {
	processor := NewProcessor(&stubAccessTokenStore{}, &stubTextMessageSender{})

	err := processor.Process(context.Background(), event.InboundMessage{
		Type: event.MessageTypeInteractive,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestProcessReturnsErrors(t *testing.T) {
	databaseError := errors.New("database unavailable")
	senderError := errors.New("Meta unavailable")
	tests := []struct {
		name        string
		message     event.InboundMessage
		tokenError  error
		senderError error
		wantError   error
	}{
		{
			name:      "missing text content",
			message:   event.InboundMessage{Type: event.MessageTypeText},
			wantError: errors.New("inbound text message has no text content"),
		},
		{
			name:       "token lookup error",
			message:    textMessage(),
			tokenError: databaseError,
			wantError:  databaseError,
		},
		{
			name:        "sender error",
			message:     textMessage(),
			senderError: senderError,
			wantError:   senderError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewProcessor(
				&stubAccessTokenStore{token: "access-token", err: tt.tokenError},
				&stubTextMessageSender{err: tt.senderError},
			)
			err := processor.Process(context.Background(), tt.message)
			if tt.wantError == databaseError || tt.wantError == senderError {
				if !errors.Is(err, tt.wantError) {
					t.Fatalf("expected error %v, got %v", tt.wantError, err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantError.Error() {
				t.Fatalf("expected error %v, got %v", tt.wantError, err)
			}
		})
	}
}

func textMessage() event.InboundMessage {
	return event.InboundMessage{
		WABAID:        "waba-id",
		PhoneNumberID: "phone-number-id",
		From:          "15551234567",
		Type:          event.MessageTypeText,
		Text:          &event.TextContent{Body: "Hello"},
	}
}
