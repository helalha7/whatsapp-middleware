package message

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/helalha7/whatsapp-middleware/internal/data"
)

func TestStoreInsert(t *testing.T) {
	createdAt := time.Date(2026, time.November, 26, 10, 0, 0, 0, time.UTC)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create mock database: %v", err)
	}
	defer db.Close()

	message := &Message{
		ConversationID: 12,
		ChatSessionID:  31,
		Direction:      DirectionOutbound,
		Content:        "Hello\nHow are you?",
	}
	mock.ExpectBegin()
	expectMessageInsert(mock, message, int64(31), nil, createdAt)
	mock.ExpectCommit()

	store := &Store{DB: db}
	if err := store.Insert(context.Background(), message); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet database expectations: %v", err)
	}
}

func TestStoreInsertTranslatesDriverErrors(t *testing.T) {
	tests := []struct {
		name      string
		driverErr error
		wantError error
	}{
		{
			name:      "duplicate WhatsApp message",
			driverErr: &mysql.MySQLError{Number: 1062, Message: "duplicate entry"},
			wantError: data.ErrDuplicateRecord,
		},
		{
			name:      "unknown session",
			driverErr: &mysql.MySQLError{Number: 1452, Message: "foreign key violation"},
			wantError: data.ErrRecordNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("create mock database: %v", err)
			}
			defer db.Close()

			mock.ExpectBegin()
			mock.ExpectExec("INSERT INTO messages").
				WithArgs(int64(12), int64(31), "wamid.message-id", DirectionInbound, "Hello").
				WillReturnError(tt.driverErr)
			mock.ExpectRollback()

			store := &Store{DB: db}
			err = store.Insert(context.Background(), &Message{
				ConversationID:    12,
				ChatSessionID:     31,
				WhatsAppMessageID: "wamid.message-id",
				Direction:         DirectionInbound,
				Content:           "Hello",
			})
			if !errors.Is(err, tt.wantError) {
				t.Fatalf("expected error %v, got %v", tt.wantError, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet database expectations: %v", err)
			}
		})
	}
}

func TestStoreListBySessionThrough(t *testing.T) {
	createdAt := time.Date(2026, time.November, 26, 10, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows(messageColumns()).
		AddRow(20, 12, 31, "wamid.message-id", DirectionInbound, "Hello", createdAt).
		AddRow(21, 12, 31, "", DirectionOutbound, "Hello", createdAt.Add(time.Second))

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create mock database: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT (.+) FROM messages WHERE chat_session_id = \\? AND id <= \\?").
		WithArgs(int64(31), int64(21)).
		WillReturnRows(rows)

	store := &Store{DB: db}
	messages, err := store.ListBySessionThrough(context.Background(), 31, 21)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assertMessageList(t, messages)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet database expectations: %v", err)
	}
}

func TestStoreListByConversation(t *testing.T) {
	createdAt := time.Date(2026, time.November, 26, 10, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows(messageColumns()).
		AddRow(20, 12, 31, "wamid.message-id", DirectionInbound, "Hello", createdAt).
		AddRow(21, 12, 31, "", DirectionOutbound, "Hello", createdAt.Add(time.Second))

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create mock database: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT (.+) FROM \\(").
		WithArgs(int64(7), int64(12), 50).
		WillReturnRows(rows)

	store := &Store{DB: db}
	messages, err := store.ListByConversation(context.Background(), 7, 12, 50)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assertMessageList(t, messages)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet database expectations: %v", err)
	}
}

func expectMessageInsert(
	mock sqlmock.Sqlmock,
	message *Message,
	chatSessionID any,
	whatsappMessageID any,
	createdAt time.Time,
) {
	mock.ExpectExec("INSERT INTO messages").
		WithArgs(
			message.ConversationID,
			chatSessionID,
			whatsappMessageID,
			message.Direction,
			message.Content,
		).
		WillReturnResult(sqlmock.NewResult(21, 1))
	mock.ExpectQuery("SELECT created_at FROM messages WHERE id = \\?").
		WithArgs(int64(21)).
		WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(createdAt))
	mock.ExpectExec("UPDATE chat_sessions SET last_activity_at = \\?").
		WithArgs(createdAt, chatSessionID, createdAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE conversations SET last_message_at = \\?").
		WithArgs(createdAt, message.ConversationID, createdAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

func assertMessageList(t *testing.T, messages []Message) {
	t.Helper()
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].Direction != DirectionInbound || messages[1].Direction != DirectionOutbound {
		t.Errorf("unexpected message order: %+v", messages)
	}
	if messages[0].ChatSessionID != 31 || messages[1].ChatSessionID != 31 {
		t.Errorf("unexpected chat session IDs: %+v", messages)
	}
}

func messageColumns() []string {
	return []string{
		"id",
		"conversation_id",
		"chat_session_id",
		"whatsapp_message_id",
		"direction",
		"content",
		"created_at",
	}
}
