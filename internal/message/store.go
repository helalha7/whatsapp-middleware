package message

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

func (s *Store) Insert(ctx context.Context, message *Message) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var whatsappMessageID any
	if message.WhatsAppMessageID != "" {
		whatsappMessageID = message.WhatsAppMessageID
	}

	query := `
		INSERT INTO messages (conversation_id, whatsapp_message_id, direction, content)
		VALUES (?, ?, ?, ?)
	`
	result, err := tx.ExecContext(
		ctx,
		query,
		message.ConversationID,
		whatsappMessageID,
		message.Direction,
		message.Content,
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			switch mysqlErr.Number {
			case 1062:
				return data.ErrDuplicateRecord
			case 1452:
				return data.ErrRecordNotFound
			}
		}
		return err
	}

	message.ID, err = result.LastInsertId()
	if err != nil {
		return err
	}

	query = `
		SELECT created_at
		FROM messages
		WHERE id = ?
	`
	if err := tx.QueryRowContext(ctx, query, message.ID).Scan(&message.CreatedAt); err != nil {
		return err
	}

	query = `
		UPDATE conversations
		SET last_message_at = ?
		WHERE id = ?
		AND (last_message_at IS NULL OR last_message_at <= ?)
	`
	if _, err := tx.ExecContext(
		ctx,
		query,
		message.CreatedAt,
		message.ConversationID,
		message.CreatedAt,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) ListByConversation(
	ctx context.Context,
	tenantID int64,
	conversationID int64,
	limit int,
) ([]Message, error) {
	query := `
		SELECT id, conversation_id, COALESCE(whatsapp_message_id, ''), direction, content, created_at
		FROM (
			SELECT m.id, m.conversation_id, m.whatsapp_message_id, m.direction, m.content, m.created_at
			FROM messages AS m
			INNER JOIN conversations AS c ON c.id = m.conversation_id
			WHERE c.tenant_id = ? AND c.id = ?
			ORDER BY m.created_at DESC, m.id DESC
			LIMIT ?
		) AS recent_messages
		ORDER BY created_at, id
	`
	rows, err := s.DB.QueryContext(ctx, query, tenantID, conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]Message, 0)
	for rows.Next() {
		var message Message
		if err := rows.Scan(
			&message.ID,
			&message.ConversationID,
			&message.WhatsAppMessageID,
			&message.Direction,
			&message.Content,
			&message.CreatedAt,
		); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}
