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

	tests := []struct {
		name              string
		message           Message
		whatsappMessageID any
	}{
		{
			name: "inbound message",
			message: Message{
				ConversationID:    12,
				WhatsAppMessageID: "wamid.message-id",
				Direction:         DirectionInbound,
				Content:           "Hello",
			},
			whatsappMessageID: "wamid.message-id",
		},
		{
			name: "outbound message without WhatsApp ID",
			message: Message{
				ConversationID: 12,
				Direction:      DirectionOutbound,
				Content:        "Hello",
			},
			whatsappMessageID: nil,
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
				WithArgs(int64(12), tt.whatsappMessageID, tt.message.Direction, "Hello").
				WillReturnResult(sqlmock.NewResult(21, 1))
			mock.ExpectQuery("SELECT created_at FROM messages WHERE id = \\?").
				WithArgs(int64(21)).
				WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(createdAt))
			mock.ExpectExec("UPDATE conversations SET last_message_at = \\?").
				WithArgs(createdAt, int64(12), createdAt).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			store := &Store{DB: db}
			err = store.Insert(context.Background(), &tt.message)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.message.ID != 21 {
				t.Errorf("expected ID 21, got %d", tt.message.ID)
			}
			if !tt.message.CreatedAt.Equal(createdAt) {
				t.Errorf("expected creation time %v, got %v", createdAt, tt.message.CreatedAt)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet database expectations: %v", err)
			}
		})
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
			name:      "unknown conversation",
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
				WithArgs(int64(12), "wamid.message-id", DirectionInbound, "Hello").
				WillReturnError(tt.driverErr)
			mock.ExpectRollback()

			store := &Store{DB: db}
			err = store.Insert(context.Background(), &Message{
				ConversationID:    12,
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

func TestStoreInsertRollsBackWhenConversationUpdateFails(t *testing.T) {
	createdAt := time.Date(2026, time.November, 26, 10, 0, 0, 0, time.UTC)
	databaseError := errors.New("database unavailable")
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create mock database: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO messages").
		WithArgs(int64(12), "wamid.message-id", DirectionInbound, "Hello").
		WillReturnResult(sqlmock.NewResult(21, 1))
	mock.ExpectQuery("SELECT created_at FROM messages WHERE id = \\?").
		WithArgs(int64(21)).
		WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(createdAt))
	mock.ExpectExec("UPDATE conversations SET last_message_at = \\?").
		WithArgs(createdAt, int64(12), createdAt).
		WillReturnError(databaseError)
	mock.ExpectRollback()

	store := &Store{DB: db}
	err = store.Insert(context.Background(), &Message{
		ConversationID:    12,
		WhatsAppMessageID: "wamid.message-id",
		Direction:         DirectionInbound,
		Content:           "Hello",
	})
	if err != databaseError {
		t.Fatalf("expected original error %v, got %v", databaseError, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet database expectations: %v", err)
	}
}

func TestStoreListByConversation(t *testing.T) {
	firstCreatedAt := time.Date(2026, time.November, 26, 10, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows(messageColumns()).
		AddRow(20, 12, "wamid.message-id", DirectionInbound, "Hello", firstCreatedAt).
		AddRow(21, 12, "", DirectionOutbound, "Hello", firstCreatedAt.Add(time.Second))

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
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].Direction != DirectionInbound || messages[1].Direction != DirectionOutbound {
		t.Errorf("unexpected message order: %+v", messages)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet database expectations: %v", err)
	}
}

func messageColumns() []string {
	return []string{
		"id",
		"conversation_id",
		"whatsapp_message_id",
		"direction",
		"content",
		"created_at",
	}
}
