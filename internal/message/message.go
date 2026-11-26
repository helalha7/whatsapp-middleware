package message

import "time"

type Direction string

const (
	DirectionInbound  Direction = "inbound"
	DirectionOutbound Direction = "outbound"
)

type Message struct {
	ID                int64     `json:"id"`
	ConversationID    int64     `json:"-"`
	ChatSessionID     int64     `json:"chat_session_id"`
	WhatsAppMessageID string    `json:"whatsapp_message_id,omitempty"`
	Direction         Direction `json:"direction"`
	Content           string    `json:"content"`
	CreatedAt         time.Time `json:"created_at"`
}
