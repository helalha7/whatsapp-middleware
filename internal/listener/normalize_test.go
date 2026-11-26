package listener

import (
	"strings"
	"testing"

	"github.com/helalha7/whatsapp-middleware/internal/event"
)

func TestNormalizePayloadTextMessages(t *testing.T) {
	p := textPayload()
	p.Entry[0].Changes[0].Value.Messages = append(
		p.Entry[0].Changes[0].Value.Messages,
		message{
			From:      "15550000002",
			ID:        "wamid.second",
			Timestamp: "1794312001",
			Type:      "text",
			Text:      &textMessage{Body: "Second"},
		},
	)
	p.Entry[0].Changes[0].Value.Contacts = append(
		p.Entry[0].Changes[0].Value.Contacts,
		contactWithName("15550000002", "Second Customer"),
	)

	events, err := normalizePayload(p)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	first := events[0]
	if first.ID != "wamid.message-id" || first.WABAID != "waba-id" {
		t.Errorf("unexpected message identity: %+v", first)
	}
	if first.PhoneNumberID != "phone-number-id" || first.From != "15551234567" {
		t.Errorf("unexpected routing fields: %+v", first)
	}
	if first.ContactName != "Customer" || first.Timestamp != 1794312000 {
		t.Errorf("unexpected contact or timestamp: %+v", first)
	}
	if first.Type != event.MessageTypeText || first.Text == nil || first.Text.Body != "Hello" {
		t.Errorf("unexpected text content: %+v", first)
	}
	if events[1].ContactName != "Second Customer" {
		t.Errorf("expected second contact name, got %q", events[1].ContactName)
	}
}

func TestNormalizePayloadInteractiveMessages(t *testing.T) {
	tests := []struct {
		name        string
		interactive interactiveMessage
		wantType    event.InteractiveType
		wantID      string
		wantTitle   string
	}{
		{
			name: "button reply",
			interactive: interactiveMessage{
				Type: "button_reply",
				ButtonReply: &struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				}{ID: "confirm", Title: "Confirm"},
			},
			wantType:  event.InteractiveButtonReply,
			wantID:    "confirm",
			wantTitle: "Confirm",
		},
		{
			name: "list reply",
			interactive: interactiveMessage{
				Type: "list_reply",
				ListReply: &struct {
					ID          string `json:"id"`
					Title       string `json:"title"`
					Description string `json:"description"`
				}{ID: "option-1", Title: "Option 1"},
			},
			wantType:  event.InteractiveListReply,
			wantID:    "option-1",
			wantTitle: "Option 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := textPayload()
			p.Entry[0].Changes[0].Value.Messages[0].Type = "interactive"
			p.Entry[0].Changes[0].Value.Messages[0].Text = nil
			p.Entry[0].Changes[0].Value.Messages[0].Interactive = &tt.interactive

			events, err := normalizePayload(p)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if len(events) != 1 || events[0].Interactive == nil {
				t.Fatalf("expected one interactive event, got %+v", events)
			}
			reply := events[0].Interactive
			if reply.Type != tt.wantType || reply.ID != tt.wantID || reply.Title != tt.wantTitle {
				t.Errorf("unexpected interactive reply: %+v", reply)
			}
		})
	}
}

func TestNormalizePayloadIgnoresUnsupportedMessages(t *testing.T) {
	p := textPayload()
	p.Entry[0].Changes[0].Value.Messages = append(
		[]message{{
			Type: "image",
		}},
		p.Entry[0].Changes[0].Value.Messages...,
	)

	events, err := normalizePayload(p)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(events) != 1 || events[0].ID != "wamid.message-id" {
		t.Fatalf("expected only the supported text message, got %+v", events)
	}
}

func TestNormalizePayloadRejectsMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name      string
		modify    func(*payload)
		wantError string
	}{
		{
			name: "WABA ID",
			modify: func(p *payload) {
				p.Entry[0].ID = ""
			},
			wantError: "missing WABA ID",
		},
		{
			name: "message ID",
			modify: func(p *payload) {
				p.Entry[0].Changes[0].Value.Messages[0].ID = ""
			},
			wantError: "missing message ID",
		},
		{
			name: "sender",
			modify: func(p *payload) {
				p.Entry[0].Changes[0].Value.Messages[0].From = ""
			},
			wantError: "missing sender",
		},
		{
			name: "phone number ID",
			modify: func(p *payload) {
				p.Entry[0].Changes[0].Value.Metadata.PhoneNumberID = ""
			},
			wantError: "missing phone number ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := textPayload()
			tt.modify(&p)

			_, err := normalizePayload(p)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("expected error containing %q, got %v", tt.wantError, err)
			}
		})
	}
}

func textPayload() payload {
	return payload{
		Object: "whatsapp_business_account",
		Entry: []entry{{
			ID: "waba-id",
			Changes: []change{{
				Field: "messages",
				Value: value{
					MessagingProduct: "whatsapp",
					Metadata: metadata{
						PhoneNumberID:      "phone-number-id",
						DisplayPhoneNumber: "15552050459",
					},
					Contacts: []contact{contactWithName("15551234567", "Customer")},
					Messages: []message{{
						From:      "15551234567",
						ID:        "wamid.message-id",
						Timestamp: "1794312000",
						Type:      "text",
						Text:      &textMessage{Body: "Hello"},
					}},
				},
			}},
		}},
	}
}

func contactWithName(waID string, name string) contact {
	contact := contact{WAID: waID}
	contact.Profile.Name = name
	return contact
}
