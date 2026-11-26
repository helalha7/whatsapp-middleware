package chatsession

import "time"

type Session struct {
	ID             int64     `json:"id"`
	ConversationID int64     `json:"conversation_id"`
	StartedAt      time.Time `json:"started_at"`
	LastActivityAt time.Time `json:"last_activity_at"`
}
