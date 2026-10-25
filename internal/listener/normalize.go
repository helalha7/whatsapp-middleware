package listener

import (
	"fmt"
	"strconv"

	"github.com/helalha7/whatsapp-middleware/internal/event"
)

func normalizePayload(
	p payload,
) ([]event.InboundMessage, error) {
	var events []event.InboundMessage

	for _, entry := range p.Entry {
		for _, change := range entry.Changes {
			value := change.Value

			for _, msg := range value.Messages {
				e := event.InboundMessage{
					ID:                 msg.ID,
					WABAID:             entry.ID,
					PhoneNumberID:      value.Metadata.PhoneNumberID,
					DisplayPhoneNumber: value.Metadata.DisplayPhoneNumber,
					From:               msg.From,
				}

				// Find the contact associated with this message.
				for _, contact := range value.Contacts {
					if contact.WAID == msg.From {
						e.ContactName = contact.Profile.Name
						break
					}
				}

				// Convert the WhatsApp timestamp to int64.
				timestamp, err := strconv.ParseInt(msg.Timestamp, 10, 64)
				if err != nil {
					return nil, fmt.Errorf(
						"invalid timestamp for message %s: %w",
						msg.ID, err,
					)
				}
				e.Timestamp = timestamp

				switch msg.Type {
				case "text":
					if msg.Text == nil {
						return nil, fmt.Errorf(
							"missing text content for message %s",
							msg.ID,
						)
					}

					e.Type = event.MessageTypeText
					e.Text = &event.TextContent{
						Body: msg.Text.Body,
					}

				case "interactive":
					if msg.Interactive == nil {
						return nil, fmt.Errorf(
							"missing interactive content for message %s",
							msg.ID,
						)
					}

					e.Type = event.MessageTypeInteractive

					switch msg.Interactive.Type {
					case "button_reply":
						if msg.Interactive.ButtonReply == nil {
							return nil, fmt.Errorf(
								"missing button reply for message %s",
								msg.ID,
							)
						}

						e.Interactive = &event.InteractiveReply{
							Type:  event.InteractiveButtonReply,
							ID:    msg.Interactive.ButtonReply.ID,
							Title: msg.Interactive.ButtonReply.Title,
						}

					case "list_reply":
						if msg.Interactive.ListReply == nil {
							return nil, fmt.Errorf(
								"missing list reply for message %s",
								msg.ID,
							)
						}

						e.Interactive = &event.InteractiveReply{
							Type:  event.InteractiveListReply,
							ID:    msg.Interactive.ListReply.ID,
							Title: msg.Interactive.ListReply.Title,
						}

					default:
						continue
					}

				default:
					// Ignore unsupported message types.
					continue
				}

				events = append(events, e)
			}
		}
	}

	return events, nil
}
