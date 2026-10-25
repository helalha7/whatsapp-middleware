package queue

import "github.com/helalha7/whatsapp-middleware/internal/event"

var InboundMessagesQueue = make(chan event.InboundMessage)
