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
