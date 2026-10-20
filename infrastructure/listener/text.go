package listener

import "time"

type NormalizedTextMessage struct {
	BusinessID    string
	PhoneNumberID string
	CustomerID    string
	CustomerName  string
	MessageID     string
	Text          string
	Timestamp     time.Time
}
