package messaging

import (
	"context"
	"errors"

	"github.com/helalha7/whatsapp-middleware/internal/event"
)

type accessTokenStore interface {
	GetAccessToken(ctx context.Context, wabaID string) (string, error)
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
	accessTokens accessTokenStore
	sender       textMessageSender
}

func NewProcessor(
	accessTokens accessTokenStore,
	sender textMessageSender,
) *Processor {
	return &Processor{
		accessTokens: accessTokens,
		sender:       sender,
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

	accessToken, err := p.accessTokens.GetAccessToken(ctx, message.WABAID)
	if err != nil {
		return err
	}

	return p.sender.SendTextMessage(
		ctx,
		message.PhoneNumberID,
		accessToken,
		message.From,
		message.Text.Body,
	)
}
