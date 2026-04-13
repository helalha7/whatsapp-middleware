package webhook

import "fmt"

type Entry struct {
	ID      string   `json:"id"`
	Changes []Change `json:"changes"`
}

type Change struct {
	Field string      `json:"field"`
	Value ChangeValue `json:"value"`
}

type ChangeValue struct {
	MessagingProduct string    `json:"messaging_product"`
	Metadata         Metadata  `json:"metadata"`
	Contacts         []Contact `json:"contacts"`
	Messages         []Message `json:"messages"`
}

type Metadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type Contact struct {
	Profile Profile `json:"profile"`
	WaID    string  `json:"wa_id"`
}

type Profile struct {
	Name string `json:"name"`
}

type Message struct {
	From      string `json:"from"`
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Text      Text   `json:"text"`
}

type Text struct {
	Body string `json:"body"`
}

type WebhookPayload struct {
	Object string  `json:"object"`
	Entry  []Entry `json:"entry"`
}

func (payload *WebhookPayload) getMessageText() string {
	return payload.Entry[0].Changes[0].Value.Messages[0].Text.Body
}

func (payload *WebhookPayload) getSenderName() string {
	return payload.Entry[0].Changes[0].Value.Contacts[0].Profile.Name
}

func (payload *WebhookPayload) getSenderPhoneNumber() string {
	return payload.Entry[0].Changes[0].Value.Metadata.DisplayPhoneNumber
}

func (payload *WebhookPayload) String() string {
	return fmt.Sprintf("{\nFrom : {name : %v, phone number: %v},\nMessage : '%v'\n}", payload.getSenderName(), payload.getSenderPhoneNumber(), payload.getMessageText())
}
