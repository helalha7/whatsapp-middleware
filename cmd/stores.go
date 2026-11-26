package main

import (
	"database/sql"

	"github.com/helalha7/whatsapp-middleware/internal/chatsession"
	"github.com/helalha7/whatsapp-middleware/internal/conversation"
	"github.com/helalha7/whatsapp-middleware/internal/idempotency"
	"github.com/helalha7/whatsapp-middleware/internal/message"
	"github.com/helalha7/whatsapp-middleware/internal/tenant"
)

type stores struct {
	tenants         *tenant.Store
	conversations   *conversation.Store
	chatSessions    *chatsession.Store
	messages        *message.Store
	inboundReceipts *idempotency.Store
}

func newStores(db *sql.DB) *stores {
	return &stores{
		tenants:         &tenant.Store{DB: db},
		conversations:   &conversation.Store{DB: db},
		chatSessions:    &chatsession.Store{DB: db},
		messages:        &message.Store{DB: db},
		inboundReceipts: &idempotency.Store{DB: db},
	}
}
