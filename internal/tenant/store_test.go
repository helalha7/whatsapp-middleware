package tenant

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/helalha7/whatsapp-middleware/internal/data"
)

func TestGetMessagingCredentials(t *testing.T) {
	databaseError := errors.New("database unavailable")
	tests := []struct {
		name       string
		rows       *sqlmock.Rows
		queryError error
		wantID     int64
		wantToken  string
		wantError  error
		exactError error
	}{
		{
			name:      "gets credentials",
			rows:      sqlmock.NewRows([]string{"id", "access_token"}).AddRow(7, "access-token"),
			wantID:    7,
			wantToken: "access-token",
		},
		{
			name:      "unknown WABA",
			rows:      sqlmock.NewRows([]string{"id", "access_token"}),
			wantError: data.ErrRecordNotFound,
		},
		{
			name:       "database error",
			queryError: databaseError,
			exactError: databaseError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("create mock database: %v", err)
			}
			defer db.Close()

			expectation := mock.ExpectQuery("SELECT id, access_token FROM tenants WHERE waba_id = \\?").
				WithArgs("waba-id")
			if tt.queryError != nil {
				expectation.WillReturnError(tt.queryError)
			} else {
				expectation.WillReturnRows(tt.rows)
			}

			store := &Store{DB: db}
			tenantID, token, err := store.GetMessagingCredentials(
				context.Background(),
				"waba-id",
			)
			if tt.wantError != nil && !errors.Is(err, tt.wantError) {
				t.Fatalf("expected error %v, got %v", tt.wantError, err)
			}
			if tt.exactError != nil && err != tt.exactError {
				t.Fatalf("expected original error %v, got %v", tt.exactError, err)
			}
			if tt.wantError == nil && tt.exactError == nil {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if tenantID != tt.wantID {
					t.Errorf("expected tenant ID %d, got %d", tt.wantID, tenantID)
				}
				if token != tt.wantToken {
					t.Errorf("expected access token %q, got %q", tt.wantToken, token)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet database expectations: %v", err)
			}
		})
	}
}
