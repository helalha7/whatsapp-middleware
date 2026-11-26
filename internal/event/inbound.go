package event

type MessageType string

const (
	MessageTypeText        MessageType = "text"
	MessageTypeInteractive MessageType = "interactive"
)

type InteractiveType string

const (
	InteractiveButtonReply InteractiveType = "button_reply"
	InteractiveListReply   InteractiveType = "list_reply"
)

type InboundMessage struct {
	WhatsAppMessageID          string
	BusinessAccountID          string
	BusinessPhoneNumberID      string
	BusinessDisplayPhoneNumber string

	CustomerWhatsAppID string
	CustomerName       string
	SentAtUnix         int64

	MessageType      MessageType
	TextContent      *TextContent
	InteractiveReply *InteractiveReply
}

type TextContent struct {
	Body string
}

type InteractiveReply struct {
	Type  InteractiveType
	ID    string
	Title string
}
