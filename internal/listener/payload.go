package listener

type payload struct {
	Object string  `json:"object"`
	Entry  []entry `json:"entry"`
}

type entry struct {
	ID      string   `json:"id"`
	Changes []change `json:"changes"`
}

type change struct {
	Value value  `json:"value"`
	Field string `json:"field"`
}

type value struct {
	MessagingProduct string    `json:"messaging_product"`
	Metadata         metadata  `json:"metadata"`
	Contacts         []contact `json:"contacts"`
	Messages         []message `json:"messages"`
}

type metadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type contact struct {
	Profile struct {
		Name string `json:"name"`
	} `json:"profile"`
	WAID string `json:"wa_id"`
}

type message struct {
	From      string `json:"from"`
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`

	Text        *textMessage        `json:"text,omitempty"`
	Interactive *interactiveMessage `json:"interactive,omitempty"`
	Image       *mediaMessage       `json:"image,omitempty"`
}

type textMessage struct {
	Body string `json:"body"`
}

type interactiveMessage struct {
	Type string `json:"type"`

	ButtonReply *struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"button_reply,omitempty"`

	ListReply *struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
	} `json:"list_reply,omitempty"`
}

type mediaMessage struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type"`
	SHA256   string `json:"sha256"`
}
