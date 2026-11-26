package conversation

import "time"

type Conversation struct {
	ID            int64      `json:"id"`
	TenantID      int64      `json:"-"`
	CustomerWAID  string     `json:"customer_wa_id"`
	CustomerName  string     `json:"customer_name"`
	LastMessageAt *time.Time `json:"last_message_at"`
	CreatedAt     time.Time  `json:"created_at"`
}
