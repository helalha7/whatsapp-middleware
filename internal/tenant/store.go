package tenant

import (
	"context"
	"database/sql"
	"errors"

	"github.com/helalha7/whatsapp-middleware/internal/data"
)

type Store struct {
	DB *sql.DB
}

func (s *Store) Get(ctx context.Context, id int) (*Tenant, error) {
	query := `
		SELECT id, created_at, version, name, waba_id, phone_number_id, display_phone_number
		FROM tenants
		WHERE id = ?
	`
	args := []any{id}
	row := s.DB.QueryRowContext(ctx, query, args...)

	var t Tenant
	err := row.Scan(
		t.ID,
		t.CreatedAt,
		t.Version,
		t.Name,
		t.WABAID,
		t.PhoneNumberID,
		t.DisplayPhoneNumber,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, data.ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &t, nil
}

func (s *Store) GetAccessToken(ctx context.Context, wabaID string) (string, error) {
	query := `
		SELECT access_token
		FROM tenants
		WHERE waba_id = ?
	`
	args := []any{wabaID}
	row := s.DB.QueryRowContext(ctx, query, args...)

	var accessToken string
	err := row.Scan(
		&accessToken,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return "", data.ErrRecordNotFound
		default:
			return "", err
		}
	}

	return accessToken, nil
}
