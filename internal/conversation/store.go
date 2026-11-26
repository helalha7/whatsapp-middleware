package conversation

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

func (s *Store) Upsert(
	ctx context.Context,
	tenantID int64,
	customerWAID string,
	customerName string,
) (*Conversation, error) {
	query := `
		INSERT INTO conversations (tenant_id, customer_wa_id, customer_name)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE customer_name = VALUES(customer_name)
	`
	_, err := s.DB.ExecContext(ctx, query, tenantID, customerWAID, customerName)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1452 {
			return nil, data.ErrRecordNotFound
		}
		return nil, err
	}

	query = `
		SELECT id, tenant_id, customer_wa_id, customer_name, last_message_at, created_at
		FROM conversations
		WHERE tenant_id = ? AND customer_wa_id = ?
	`
	return scanConversation(s.DB.QueryRowContext(ctx, query, tenantID, customerWAID))
}

func (s *Store) Get(
	ctx context.Context,
	tenantID int64,
	conversationID int64,
) (*Conversation, error) {
	query := `
		SELECT id, tenant_id, customer_wa_id, customer_name, last_message_at, created_at
		FROM conversations
		WHERE tenant_id = ? AND id = ?
	`
	return scanConversation(s.DB.QueryRowContext(ctx, query, tenantID, conversationID))
}

func (s *Store) ListByTenant(
	ctx context.Context,
	tenantID int64,
	limit int,
) ([]Conversation, error) {
	query := `
		SELECT id, tenant_id, customer_wa_id, customer_name, last_message_at, created_at
		FROM conversations
		WHERE tenant_id = ?
		ORDER BY COALESCE(last_message_at, created_at) DESC, id DESC
		LIMIT ?
	`
	rows, err := s.DB.QueryContext(ctx, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conversations := make([]Conversation, 0)
	for rows.Next() {
		conversation, err := scanConversation(rows)
		if err != nil {
			return nil, err
		}
		conversations = append(conversations, *conversation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return conversations, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanConversation(row scanner) (*Conversation, error) {
	var conversation Conversation
	err := row.Scan(
		&conversation.ID,
		&conversation.TenantID,
		&conversation.CustomerWAID,
		&conversation.CustomerName,
		&conversation.LastMessageAt,
		&conversation.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, data.ErrRecordNotFound
		}
		return nil, err
	}

	return &conversation, nil
}
