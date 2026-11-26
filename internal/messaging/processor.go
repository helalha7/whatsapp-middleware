package messaging

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/helalha7/whatsapp-middleware/internal/chatsession"
	"github.com/helalha7/whatsapp-middleware/internal/conversation"
	"github.com/helalha7/whatsapp-middleware/internal/event"
	messagedomain "github.com/helalha7/whatsapp-middleware/internal/message"
)

type tenantCredentialsStore interface {
	GetMessagingCredentials(ctx context.Context, wabaID string) (int64, string, error)
}

type conversationStore interface {
	Upsert(
		ctx context.Context,
		tenantID int64,
		customerWAID string,
		customerName string,
	) (*conversation.Conversation, error)
}

type chatSessionStore interface {
	StartOrContinue(
		ctx context.Context,
		conversationID int64,
		idleTimeout time.Duration,
	) (*chatsession.Session, error)
}

type messageStore interface {
	Insert(ctx context.Context, message *messagedomain.Message) error
	ListBySessionThrough(
		ctx context.Context,
		chatSessionID int64,
		messageID int64,
	) ([]messagedomain.Message, error)
}

type textMessageSender interface {
	SendTextMessage(
		ctx context.Context,
		senderPhoneNumberID string,
		senderAccessToken string,
		receiverPhoneNumber string,
		message string,
	) error
}

type Processor struct {
	tenants            tenantCredentialsStore
	conversations      conversationStore
	chatSessions       chatSessionStore
	messages           messageStore
	sender             textMessageSender
	sessionIdleTimeout time.Duration
}

func NewProcessor(
	tenants tenantCredentialsStore,
	conversations conversationStore,
	chatSessions chatSessionStore,
	messages messageStore,
	sender textMessageSender,
	sessionIdleTimeout time.Duration,
) *Processor {
	return &Processor{
		tenants:            tenants,
		conversations:      conversations,
		chatSessions:       chatSessions,
		messages:           messages,
		sender:             sender,
		sessionIdleTimeout: sessionIdleTimeout,
	}
}

func (p *Processor) Process(
	ctx context.Context,
	message event.InboundMessage,
) error {
	if message.Type != event.MessageTypeText {
		return nil
	}
	if message.Text == nil {
		return errors.New("inbound text message has no text content")
	}

	tenantID, accessToken, err := p.tenants.GetMessagingCredentials(ctx, message.WABAID)
	if err != nil {
		return err
	}

	conversation, err := p.conversations.Upsert(
		ctx,
		tenantID,
		message.From,
		message.ContactName,
	)
	if err != nil {
		return err
	}
	chatSession, err := p.chatSessions.StartOrContinue(
		ctx,
		conversation.ID,
		p.sessionIdleTimeout,
	)
	if err != nil {
		return err
	}

	inboundMessage := &messagedomain.Message{
		ConversationID:    conversation.ID,
		ChatSessionID:     chatSession.ID,
		WhatsAppMessageID: message.ID,
		Direction:         messagedomain.DirectionInbound,
		Content:           message.Text.Body,
	}
	if err := p.messages.Insert(ctx, inboundMessage); err != nil {
		return err
	}

	sessionMessages, err := p.messages.ListBySessionThrough(
		ctx,
		chatSession.ID,
		inboundMessage.ID,
	)
	if err != nil {
		return err
	}
	echo := buildSessionEcho(sessionMessages)

	if err := p.sender.SendTextMessage(
		ctx,
		message.PhoneNumberID,
		accessToken,
		message.From,
		echo,
	); err != nil {
		return err
	}

	return p.messages.Insert(ctx, &messagedomain.Message{
		ConversationID: conversation.ID,
		ChatSessionID:  chatSession.ID,
		Direction:      messagedomain.DirectionOutbound,
		Content:        echo,
	})
}

func buildSessionEcho(messages []messagedomain.Message) string {
	contents := make([]string, 0, len(messages))
	for _, message := range messages {
		if message.Direction == messagedomain.DirectionInbound {
			contents = append(contents, message.Content)
		}
	}
	return strings.Join(contents, "\n")
}
