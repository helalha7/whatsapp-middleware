package idempotency

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-sql-driver/mysql"
	"github.com/helalha7/whatsapp-middleware/internal/data"
)

type Store struct {
	DB *sql.DB
}

// RegisterInboundMessage records a WhatsApp message as received. It returns
// true for the first delivery and false when the message was already received.
func (s *Store) RegisterInboundMessage(
	ctx context.Context,
	wabaID string,
	messageID string,
) (bool, error) {
	query := `
		INSERT INTO received_inbound_messages (tenant_id, whatsapp_message_id)
		SELECT id, ?
		FROM tenants
		WHERE waba_id = ?
	`

	result, err := s.DB.ExecContext(ctx, query, messageID, wabaID)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return false, nil
		}

		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if rowsAffected == 0 {
		return false, data.ErrRecordNotFound
	}

	return true, nil
}
