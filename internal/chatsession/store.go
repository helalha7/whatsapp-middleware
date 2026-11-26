package chatsession

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/helalha7/whatsapp-middleware/internal/data"
)

type Store struct {
	DB *sql.DB
}

func (s *Store) StartOrContinue(
	ctx context.Context,
	conversationID int64,
	idleTimeout time.Duration,
) (*Session, error) {
	if idleTimeout <= 0 {
		return nil, errors.New("chat session idle timeout must be positive")
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		SELECT id
		FROM conversations
		WHERE id = ?
		FOR UPDATE
	`
	var lockedConversationID int64
	if err := tx.QueryRowContext(ctx, query, conversationID).Scan(&lockedConversationID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, data.ErrRecordNotFound
		}
		return nil, err
	}

	chatSessionID, err := findActive(ctx, tx, conversationID, idleTimeout)
	if err != nil {
		return nil, err
	}
	if chatSessionID == 0 {
		chatSessionID, err = insert(ctx, tx, conversationID)
		if err != nil {
			return nil, err
		}
	}

	query = `
		UPDATE chat_sessions
		SET last_activity_at = CURRENT_TIMESTAMP(6)
		WHERE id = ?
	`
	if _, err := tx.ExecContext(ctx, query, chatSessionID); err != nil {
		return nil, err
	}

	query = `
		SELECT id, conversation_id, started_at, last_activity_at
		FROM chat_sessions
		WHERE id = ?
	`
	var session Session
	if err := tx.QueryRowContext(ctx, query, chatSessionID).Scan(
		&session.ID,
		&session.ConversationID,
		&session.StartedAt,
		&session.LastActivityAt,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &session, nil
}

func findActive(
	ctx context.Context,
	tx *sql.Tx,
	conversationID int64,
	idleTimeout time.Duration,
) (int64, error) {
	query := `
		SELECT id
		FROM chat_sessions
		WHERE conversation_id = ?
		AND last_activity_at > CURRENT_TIMESTAMP(6) - INTERVAL ? MICROSECOND
		ORDER BY last_activity_at DESC, id DESC
		LIMIT 1
	`
	var chatSessionID int64
	err := tx.QueryRowContext(
		ctx,
		query,
		conversationID,
		idleTimeout.Microseconds(),
	).Scan(&chatSessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return chatSessionID, nil
}

func insert(
	ctx context.Context,
	tx *sql.Tx,
	conversationID int64,
) (int64, error) {
	query := `
		INSERT INTO chat_sessions (conversation_id)
		VALUES (?)
	`
	result, err := tx.ExecContext(ctx, query, conversationID)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1452 {
			return 0, data.ErrRecordNotFound
		}
		return 0, err
	}

	chatSessionID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return chatSessionID, nil
}
