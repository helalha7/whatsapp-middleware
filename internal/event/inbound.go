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
	ID                 string
	WABAID             string
	PhoneNumberID      string
	DisplayPhoneNumber string

	From        string
	ContactName string
	Timestamp   int64

	Type        MessageType
	Text        *TextContent
	Interactive *InteractiveReply
}

type TextContent struct {
	Body string
}

type InteractiveReply struct {
	Type  InteractiveType
	ID    string
	Title string
}
