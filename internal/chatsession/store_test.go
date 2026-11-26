package chatsession

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/helalha7/whatsapp-middleware/internal/data"
)

const testIdleTimeout = 10 * time.Minute

func TestStoreStartOrContinue(t *testing.T) {
	startedAt := time.Date(2026, time.November, 26, 10, 0, 0, 0, time.UTC)
	lastActivityAt := startedAt.Add(time.Minute)
	tests := []struct {
		name          string
		activeSession bool
	}{
		{name: "continues active session", activeSession: true},
		{name: "starts session after inactivity"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("create mock database: %v", err)
			}
			defer db.Close()

			mock.ExpectBegin()
			mock.ExpectQuery("SELECT id FROM conversations WHERE id = \\? FOR UPDATE").
				WithArgs(int64(12)).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(12))

			activeRows := sqlmock.NewRows([]string{"id"})
			if tt.activeSession {
				activeRows.AddRow(31)
			}
			mock.ExpectQuery("SELECT id FROM chat_sessions WHERE conversation_id = \\? AND last_activity_at > CURRENT_TIMESTAMP\\(6\\) - INTERVAL \\? MICROSECOND").
				WithArgs(int64(12), testIdleTimeout.Microseconds()).
				WillReturnRows(activeRows)
			if !tt.activeSession {
				mock.ExpectExec("INSERT INTO chat_sessions").
					WithArgs(int64(12)).
					WillReturnResult(sqlmock.NewResult(31, 1))
			}

			mock.ExpectExec("UPDATE chat_sessions SET last_activity_at = CURRENT_TIMESTAMP\\(6\\) WHERE id = \\?").
				WithArgs(int64(31)).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectQuery("SELECT id, conversation_id, started_at, last_activity_at FROM chat_sessions WHERE id = \\?").
				WithArgs(int64(31)).
				WillReturnRows(sqlmock.NewRows([]string{
					"id", "conversation_id", "started_at", "last_activity_at",
				}).AddRow(31, 12, startedAt, lastActivityAt))
			mock.ExpectCommit()

			store := &Store{DB: db}
			session, err := store.StartOrContinue(
				context.Background(),
				12,
				testIdleTimeout,
			)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if session.ID != 31 || session.ConversationID != 12 {
				t.Errorf("unexpected session: %+v", session)
			}
			if !session.StartedAt.Equal(startedAt) || !session.LastActivityAt.Equal(lastActivityAt) {
				t.Errorf("unexpected session timestamps: %+v", session)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet database expectations: %v", err)
			}
		})
	}
}

func TestStoreStartOrContinueRejectsInvalidTimeout(t *testing.T) {
	store := &Store{}
	_, err := store.StartOrContinue(context.Background(), 12, 0)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestStoreStartOrContinueUnknownConversation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create mock database: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM conversations WHERE id = \\? FOR UPDATE").
		WithArgs(int64(12)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectRollback()

	store := &Store{DB: db}
	_, err = store.StartOrContinue(context.Background(), 12, testIdleTimeout)
	if !errors.Is(err, data.ErrRecordNotFound) {
		t.Fatalf("expected error %v, got %v", data.ErrRecordNotFound, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet database expectations: %v", err)
	}
}
