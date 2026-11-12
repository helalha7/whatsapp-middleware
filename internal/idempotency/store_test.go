package idempotency

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/helalha7/whatsapp-middleware/internal/data"
)

func TestRegisterInboundMessage(t *testing.T) {
	tests := []struct {
		name      string
		execError error
		rows      int64
		wantNew   bool
		wantError bool
		errorIs   error
	}{
		{
			name:    "first delivery",
			rows:    1,
			wantNew: true,
		},
		{
			name:      "duplicate delivery",
			execError: &mysql.MySQLError{Number: 1062, Message: "duplicate entry"},
			wantNew:   false,
		},
		{
			name:      "database error",
			execError: errors.New("database unavailable"),
			wantError: true,
		},
		{
			name:      "unknown tenant",
			wantError: true,
			errorIs:   data.ErrRecordNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("create mock database: %v", err)
			}
			defer db.Close()

			expectation := mock.ExpectExec("INSERT INTO received_inbound_messages").
				WithArgs("wamid.message-id", "2161887421343402")
			if tt.execError != nil {
				expectation.WillReturnError(tt.execError)
			} else {
				expectation.WillReturnResult(sqlmock.NewResult(0, tt.rows))
			}

			store := &Store{DB: db}
			isNew, err := store.RegisterInboundMessage(
				context.Background(),
				"2161887421343402",
				"wamid.message-id",
			)

			if tt.wantError && err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.errorIs != nil && !errors.Is(err, tt.errorIs) {
				t.Fatalf("expected error %v, got %v", tt.errorIs, err)
			}
			if tt.wantError && tt.errorIs == nil && err != tt.execError {
				t.Fatalf("expected original database error, got %v", err)
			}
			if isNew != tt.wantNew {
				t.Errorf("expected isNew %t, got %t", tt.wantNew, isNew)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet database expectations: %v", err)
			}
		})
	}
}
