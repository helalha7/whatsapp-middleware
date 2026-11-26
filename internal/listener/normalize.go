package listener

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/helalha7/whatsapp-middleware/internal/event"
)

func normalizePayload(p payload) ([]event.InboundMessage, error) {
	var events []event.InboundMessage

	for _, entry := range p.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}

			value := change.Value
			if value.MessagingProduct != "whatsapp" {
				return nil, fmt.Errorf(
					"unexpected messaging product %q",
					value.MessagingProduct,
				)
			}
			if len(value.Messages) > 0 && entry.ID == "" {
				return nil, errors.New("missing WABA ID")
			}

			contactNames := indexContactNames(value.Contacts)
			for _, message := range value.Messages {
				normalized, supported, err := normalizeMessage(
					entry.ID,
					value.Metadata,
					contactNames,
					message,
				)
				if err != nil {
					return nil, err
				}
				if supported {
					events = append(events, normalized)
				}
			}
		}
	}

	return events, nil
}

func normalizeMessage(
	wabaID string,
	metadata metadata,
	contactNames map[string]string,
	msg message,
) (event.InboundMessage, bool, error) {
	if msg.Type != "text" && msg.Type != "interactive" {
		return event.InboundMessage{}, false, nil
	}
	if msg.ID == "" {
		return event.InboundMessage{}, false, errors.New("missing message ID")
	}
	if msg.From == "" {
		return event.InboundMessage{}, false, fmt.Errorf(
			"missing sender for message %s",
			msg.ID,
		)
	}
	if metadata.PhoneNumberID == "" {
		return event.InboundMessage{}, false, fmt.Errorf(
			"missing phone number ID for message %s",
			msg.ID,
		)
	}

	timestamp, err := strconv.ParseInt(msg.Timestamp, 10, 64)
	if err != nil {
		return event.InboundMessage{}, false, fmt.Errorf(
			"invalid timestamp for message %s: %w",
			msg.ID,
			err,
		)
	}

	normalized := event.InboundMessage{
		WhatsAppMessageID:          msg.ID,
		BusinessAccountID:          wabaID,
		BusinessPhoneNumberID:      metadata.PhoneNumberID,
		BusinessDisplayPhoneNumber: metadata.DisplayPhoneNumber,
		CustomerWhatsAppID:         msg.From,
		CustomerName:               contactNames[msg.From],
		SentAtUnix:                 timestamp,
	}

	switch msg.Type {
	case "text":
		if err := normalizeText(&normalized, msg); err != nil {
			return event.InboundMessage{}, false, err
		}
	case "interactive":
		supported, err := normalizeInteractive(&normalized, msg)
		if err != nil || !supported {
			return event.InboundMessage{}, supported, err
		}
	}

	return normalized, true, nil
}

func normalizeText(normalized *event.InboundMessage, msg message) error {
	if msg.Text == nil {
		return fmt.Errorf("missing text content for message %s", msg.ID)
	}

	normalized.MessageType = event.MessageTypeText
	normalized.TextContent = &event.TextContent{Body: msg.Text.Body}
	return nil
}

func normalizeInteractive(
	normalized *event.InboundMessage,
	msg message,
) (bool, error) {
	if msg.Interactive == nil {
		return false, fmt.Errorf(
			"missing interactive content for message %s",
			msg.ID,
		)
	}

	normalized.MessageType = event.MessageTypeInteractive
	switch msg.Interactive.Type {
	case "button_reply":
		if msg.Interactive.ButtonReply == nil {
			return false, fmt.Errorf(
				"missing button reply for message %s",
				msg.ID,
			)
		}
		normalized.InteractiveReply = &event.InteractiveReply{
			Type:  event.InteractiveButtonReply,
			ID:    msg.Interactive.ButtonReply.ID,
			Title: msg.Interactive.ButtonReply.Title,
		}
	case "list_reply":
		if msg.Interactive.ListReply == nil {
			return false, fmt.Errorf(
				"missing list reply for message %s",
				msg.ID,
			)
		}
		normalized.InteractiveReply = &event.InteractiveReply{
			Type:  event.InteractiveListReply,
			ID:    msg.Interactive.ListReply.ID,
			Title: msg.Interactive.ListReply.Title,
		}
	default:
		return false, nil
	}

	return true, nil
}

func indexContactNames(contacts []contact) map[string]string {
	names := make(map[string]string, len(contacts))
	for _, contact := range contacts {
		names[contact.WAID] = contact.Profile.Name
	}
	return names
}
