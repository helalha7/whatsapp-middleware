package messaging

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/helalha7/whatsapp-middleware/internal/chatsession"
	"github.com/helalha7/whatsapp-middleware/internal/conversation"
	"github.com/helalha7/whatsapp-middleware/internal/event"
	messagedomain "github.com/helalha7/whatsapp-middleware/internal/message"
)

type stubTenantCredentialsStore struct {
	tenantID int64
	token    string
	err      error
	wabaID   string
}

func (s *stubTenantCredentialsStore) GetMessagingCredentials(
	_ context.Context,
	wabaID string,
) (int64, string, error) {
	s.wabaID = wabaID
	return s.tenantID, s.token, s.err
}

type stubConversationStore struct {
	conversationID int64
	tenantID       int64
	customerWAID   string
	customerName   string
	err            error
}

func (s *stubConversationStore) Upsert(
	_ context.Context,
	tenantID int64,
	customerWAID string,
	customerName string,
) (*conversation.Conversation, error) {
	s.tenantID = tenantID
	s.customerWAID = customerWAID
	s.customerName = customerName
	if s.err != nil {
		return nil, s.err
	}
	return &conversation.Conversation{ID: s.conversationID}, nil
}

type stubChatSessionStore struct {
	conversationID int64
	idleTimeout    time.Duration
	sessionID      int64
	err            error
}

func (s *stubChatSessionStore) StartOrContinue(
	_ context.Context,
	conversationID int64,
	idleTimeout time.Duration,
) (*chatsession.Session, error) {
	s.conversationID = conversationID
	s.idleTimeout = idleTimeout
	if s.err != nil {
		return nil, s.err
	}
	return &chatsession.Session{ID: s.sessionID, ConversationID: conversationID}, nil
}

type stubMessageStore struct {
	messages        []messagedomain.Message
	messageID       int64
	listedSessionID int64
	throughID       int64
	inboundError    error
	listError       error
	outboundError   error
}

func (s *stubMessageStore) Insert(
	_ context.Context,
	message *messagedomain.Message,
) error {
	if message.Direction == messagedomain.DirectionInbound {
		message.ID = s.messageID
	}
	s.messages = append(s.messages, *message)
	if message.Direction == messagedomain.DirectionInbound {
		return s.inboundError
	}
	return s.outboundError
}

func (s *stubMessageStore) ListBySessionThrough(
	_ context.Context,
	chatSessionID int64,
	messageID int64,
) ([]messagedomain.Message, error) {
	s.listedSessionID = chatSessionID
	s.throughID = messageID
	return s.messages, s.listError
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
	tenants := &stubTenantCredentialsStore{tenantID: 7, token: "access-token"}
	conversations := &stubConversationStore{conversationID: 12}
	chatSessions := &stubChatSessionStore{sessionID: 31}
	messages := &stubMessageStore{
		messageID: 21,
		messages: []messagedomain.Message{
			{ChatSessionID: 31, Direction: messagedomain.DirectionInbound, Content: "First message"},
			{ChatSessionID: 31, Direction: messagedomain.DirectionOutbound, Content: "First message"},
		},
	}
	sender := &stubTextMessageSender{}
	processor := NewProcessor(tenants, conversations, chatSessions, messages, sender, 10*time.Minute)

	err := processor.Process(context.Background(), event.InboundMessage{
		ID:            "wamid.message-id",
		WABAID:        "waba-id",
		PhoneNumberID: "phone-number-id",
		From:          "15551234567",
		ContactName:   "Ada Lovelace",
		Type:          event.MessageTypeText,
		Text:          &event.TextContent{Body: "Hello"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tenants.wabaID != "waba-id" {
		t.Errorf("expected WABA ID waba-id, got %q", tenants.wabaID)
	}
	if conversations.tenantID != 7 {
		t.Errorf("expected tenant ID 7, got %d", conversations.tenantID)
	}
	if conversations.customerWAID != "15551234567" {
		t.Errorf("expected customer WA ID 15551234567, got %q", conversations.customerWAID)
	}
	if conversations.customerName != "Ada Lovelace" {
		t.Errorf("expected customer name Ada Lovelace, got %q", conversations.customerName)
	}
	if chatSessions.conversationID != 12 || chatSessions.idleTimeout != 10*time.Minute {
		t.Errorf("unexpected session selection conversation=%d timeout=%v", chatSessions.conversationID, chatSessions.idleTimeout)
	}
	if messages.listedSessionID != 31 {
		t.Errorf("expected listed session ID 31, got %d", messages.listedSessionID)
	}
	if messages.throughID != 21 {
		t.Errorf("expected session history through message 21, got %d", messages.throughID)
	}
	if len(messages.messages) != 4 {
		t.Fatalf("expected 4 session messages, got %d", len(messages.messages))
	}
	inbound := messages.messages[2]
	if inbound.ConversationID != 12 || inbound.ChatSessionID != 31 || inbound.WhatsAppMessageID != "wamid.message-id" || inbound.Direction != messagedomain.DirectionInbound || inbound.Content != "Hello" {
		t.Errorf("unexpected inbound message: %+v", inbound)
	}
	outbound := messages.messages[3]
	if outbound.ConversationID != 12 || outbound.ChatSessionID != 31 || outbound.WhatsAppMessageID != "" || outbound.Direction != messagedomain.DirectionOutbound || outbound.Content != "First message\nHello" {
		t.Errorf("unexpected outbound message: %+v", outbound)
	}
	if sender.senderPhoneNumberID != "phone-number-id" {
		t.Errorf("unexpected sender phone number ID %q", sender.senderPhoneNumberID)
	}
	if sender.senderAccessToken != "access-token" {
		t.Errorf("unexpected access token %q", sender.senderAccessToken)
	}
	if sender.receiverPhoneNumber != "15551234567" || sender.message != "First message\nHello" {
		t.Errorf("unexpected echoed message recipient=%q body=%q", sender.receiverPhoneNumber, sender.message)
	}
}

func TestProcessIgnoresUnsupportedMessageType(t *testing.T) {
	processor := NewProcessor(
		&stubTenantCredentialsStore{},
		&stubConversationStore{},
		&stubChatSessionStore{},
		&stubMessageStore{},
		&stubTextMessageSender{},
		10*time.Minute,
	)

	err := processor.Process(context.Background(), event.InboundMessage{
		Type: event.MessageTypeInteractive,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestProcessReturnsErrors(t *testing.T) {
	databaseError := errors.New("database unavailable")
	conversationError := errors.New("conversation unavailable")
	chatSessionError := errors.New("chat session unavailable")
	inboundMessageError := errors.New("inbound message unavailable")
	messageListError := errors.New("message list unavailable")
	outboundMessageError := errors.New("outbound message unavailable")
	senderError := errors.New("Meta unavailable")
	tests := []struct {
		name              string
		message           event.InboundMessage
		tenantError       error
		conversationError error
		chatSessionError  error
		inboundError      error
		listError         error
		outboundError     error
		senderError       error
		wantError         error
	}{
		{
			name:      "missing text content",
			message:   event.InboundMessage{Type: event.MessageTypeText},
			wantError: errors.New("inbound text message has no text content"),
		},
		{
			name:        "tenant lookup error",
			message:     textMessage(),
			tenantError: databaseError,
			wantError:   databaseError,
		},
		{
			name:              "conversation upsert error",
			message:           textMessage(),
			conversationError: conversationError,
			wantError:         conversationError,
		},
		{
			name:             "chat session error",
			message:          textMessage(),
			chatSessionError: chatSessionError,
			wantError:        chatSessionError,
		},
		{
			name:         "inbound message insert error",
			message:      textMessage(),
			inboundError: inboundMessageError,
			wantError:    inboundMessageError,
		},
		{
			name:      "session message list error",
			message:   textMessage(),
			listError: messageListError,
			wantError: messageListError,
		},
		{
			name:        "sender error",
			message:     textMessage(),
			senderError: senderError,
			wantError:   senderError,
		},
		{
			name:          "outbound message insert error",
			message:       textMessage(),
			outboundError: outboundMessageError,
			wantError:     outboundMessageError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewProcessor(
				&stubTenantCredentialsStore{tenantID: 7, token: "access-token", err: tt.tenantError},
				&stubConversationStore{conversationID: 12, err: tt.conversationError},
				&stubChatSessionStore{sessionID: 31, err: tt.chatSessionError},
				&stubMessageStore{messageID: 21, inboundError: tt.inboundError, listError: tt.listError, outboundError: tt.outboundError},
				&stubTextMessageSender{err: tt.senderError},
				10*time.Minute,
			)
			err := processor.Process(context.Background(), tt.message)
			if tt.wantError == databaseError || tt.wantError == conversationError || tt.wantError == chatSessionError || tt.wantError == inboundMessageError || tt.wantError == messageListError || tt.wantError == senderError || tt.wantError == outboundMessageError {
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
		ID:            "wamid.message-id",
		WABAID:        "waba-id",
		PhoneNumberID: "phone-number-id",
		From:          "15551234567",
		Type:          event.MessageTypeText,
		Text:          &event.TextContent{Body: "Hello"},
	}
}
