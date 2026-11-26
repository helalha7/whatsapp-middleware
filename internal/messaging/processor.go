package messaging

import (
	"context"
	"errors"

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

type messageStore interface {
	Insert(ctx context.Context, message *messagedomain.Message) error
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
	tenants       tenantCredentialsStore
	conversations conversationStore
	messages      messageStore
	sender        textMessageSender
}

func NewProcessor(
	tenants tenantCredentialsStore,
	conversations conversationStore,
	messages messageStore,
	sender textMessageSender,
) *Processor {
	return &Processor{
		tenants:       tenants,
		conversations: conversations,
		messages:      messages,
		sender:        sender,
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
	if err := p.messages.Insert(ctx, &messagedomain.Message{
		ConversationID:    conversation.ID,
		WhatsAppMessageID: message.ID,
		Direction:         messagedomain.DirectionInbound,
		Content:           message.Text.Body,
	}); err != nil {
		return err
	}

	if err := p.sender.SendTextMessage(
		ctx,
		message.PhoneNumberID,
		accessToken,
		message.From,
		message.Text.Body,
	); err != nil {
		return err
	}

	return p.messages.Insert(ctx, &messagedomain.Message{
		ConversationID: conversation.ID,
		Direction:      messagedomain.DirectionOutbound,
		Content:        message.Text.Body,
	})
}
