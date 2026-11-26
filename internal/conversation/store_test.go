package conversation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/helalha7/whatsapp-middleware/internal/data"
)

func TestStoreUpsert(t *testing.T) {
	createdAt := time.Date(2026, time.November, 26, 10, 0, 0, 0, time.UTC)
	databaseError := errors.New("database unavailable")

	tests := []struct {
		name           string
		customerName   string
		execError      error
		queryRows      *sqlmock.Rows
		wantID         int64
		wantName       string
		wantError      error
		wantExactError error
	}{
		{
			name:         "creates conversation",
			customerName: "Ada Lovelace",
			queryRows: sqlmock.NewRows(conversationColumns()).AddRow(
				1, 7, "15551234567", "Ada Lovelace", nil, createdAt,
			),
			wantID:   1,
			wantName: "Ada Lovelace",
		},
		{
			name:         "returns existing conversation with updated name",
			customerName: "Ada Byron",
			queryRows: sqlmock.NewRows(conversationColumns()).AddRow(
				1, 7, "15551234567", "Ada Byron", nil, createdAt,
			),
			wantID:   1,
			wantName: "Ada Byron",
		},
		{
			name:         "translates unknown tenant",
			customerName: "Ada Lovelace",
			execError:    &mysql.MySQLError{Number: 1452, Message: "foreign key violation"},
			wantError:    data.ErrRecordNotFound,
		},
		{
			name:           "returns insert error",
			customerName:   "Ada Lovelace",
			execError:      databaseError,
			wantExactError: databaseError,
		},
		{
			name:         "returns lookup error",
			customerName: "Ada Lovelace",
			queryRows:    sqlmock.NewRows(conversationColumns()),
			wantError:    data.ErrRecordNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("create mock database: %v", err)
			}
			defer db.Close()

			execExpectation := mock.ExpectExec("INSERT INTO conversations").
				WithArgs(int64(7), "15551234567", tt.customerName)
			if tt.execError != nil {
				execExpectation.WillReturnError(tt.execError)
			} else {
				execExpectation.WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectQuery("SELECT (.+) FROM conversations WHERE tenant_id = \\? AND customer_wa_id = \\?").
					WithArgs(int64(7), "15551234567").
					WillReturnRows(tt.queryRows)
			}

			store := &Store{DB: db}
			conversation, err := store.Upsert(
				context.Background(),
				7,
				"15551234567",
				tt.customerName,
			)

			if tt.wantError != nil && !errors.Is(err, tt.wantError) {
				t.Fatalf("expected error %v, got %v", tt.wantError, err)
			}
			if tt.wantExactError != nil && err != tt.wantExactError {
				t.Fatalf("expected original error %v, got %v", tt.wantExactError, err)
			}
			if tt.wantError == nil && tt.wantExactError == nil {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if conversation.ID != tt.wantID {
					t.Errorf("expected ID %d, got %d", tt.wantID, conversation.ID)
				}
				if conversation.CustomerName != tt.wantName {
					t.Errorf("expected customer name %q, got %q", tt.wantName, conversation.CustomerName)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet database expectations: %v", err)
			}
		})
	}
}

func TestStoreGet(t *testing.T) {
	createdAt := time.Date(2026, time.November, 26, 10, 0, 0, 0, time.UTC)
	lastMessageAt := createdAt.Add(time.Hour)

	tests := []struct {
		name      string
		rows      *sqlmock.Rows
		wantError error
	}{
		{
			name: "gets tenant conversation",
			rows: sqlmock.NewRows(conversationColumns()).AddRow(
				12, 7, "15551234567", "Ada Lovelace", lastMessageAt, createdAt,
			),
		},
		{
			name:      "does not return another tenant conversation",
			rows:      sqlmock.NewRows(conversationColumns()),
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

			mock.ExpectQuery("SELECT (.+) FROM conversations WHERE tenant_id = \\? AND id = \\?").
				WithArgs(int64(7), int64(12)).
				WillReturnRows(tt.rows)

			store := &Store{DB: db}
			conversation, err := store.Get(context.Background(), 7, 12)
			if !errors.Is(err, tt.wantError) {
				t.Fatalf("expected error %v, got %v", tt.wantError, err)
			}
			if tt.wantError == nil {
				if conversation.TenantID != 7 {
					t.Errorf("expected tenant ID 7, got %d", conversation.TenantID)
				}
				if conversation.LastMessageAt == nil || !conversation.LastMessageAt.Equal(lastMessageAt) {
					t.Errorf("expected last message time %v, got %v", lastMessageAt, conversation.LastMessageAt)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet database expectations: %v", err)
			}
		})
	}
}

func TestStoreListByTenant(t *testing.T) {
	createdAt := time.Date(2026, time.November, 26, 10, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows(conversationColumns()).
		AddRow(2, 7, "15550000002", "Grace Hopper", createdAt.Add(time.Hour), createdAt).
		AddRow(1, 7, "15550000001", "Ada Lovelace", nil, createdAt)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create mock database: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT (.+) FROM conversations WHERE tenant_id = \\? ORDER BY").
		WithArgs(int64(7), 50).
		WillReturnRows(rows)

	store := &Store{DB: db}
	conversations, err := store.ListByTenant(context.Background(), 7, 50)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(conversations) != 2 {
		t.Fatalf("expected 2 conversations, got %d", len(conversations))
	}
	if conversations[0].ID != 2 || conversations[1].ID != 1 {
		t.Errorf("unexpected conversation order: %+v", conversations)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet database expectations: %v", err)
	}
}

func TestStoreListByTenantReturnsDatabaseError(t *testing.T) {
	databaseError := errors.New("database unavailable")
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create mock database: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT (.+) FROM conversations WHERE tenant_id = \\? ORDER BY").
		WithArgs(int64(7), 50).
		WillReturnError(databaseError)

	store := &Store{DB: db}
	_, err = store.ListByTenant(context.Background(), 7, 50)
	if err != databaseError {
		t.Fatalf("expected original error %v, got %v", databaseError, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet database expectations: %v", err)
	}
}

func conversationColumns() []string {
	return []string{
		"id",
		"tenant_id",
		"customer_wa_id",
		"customer_name",
		"last_message_at",
		"created_at",
	}
}
