package tenant

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

func (s *Store) Insert(ctx context.Context, tenant *Tenant) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO tenants (name, waba_id, phone_number_id, display_phone_number, access_token)
		VALUES (?, ?, ?, ?, ?)
	`
	args := []any{
		tenant.Name,
		tenant.WABAID,
		tenant.PhoneNumberID,
		tenant.DisplayPhoneNumber,
		tenant.AccessToken,
	}

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return data.ErrDuplicateRecord
		}
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	query = `
		SELECT id, created_at, version
		FROM tenants
		WHERE id = ?
	`
	err = tx.QueryRowContext(ctx, query, id).Scan(
		&tenant.ID,
		&tenant.CreatedAt,
		&tenant.Version,
	)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (s *Store) Get(
	ctx context.Context,
	id int,
) (*Tenant, error) {
	query := `
		SELECT id, created_at, version, name, waba_id, phone_number_id, display_phone_number
		FROM tenants
		WHERE id = ?
	`
	args := []any{id}
	row := s.DB.QueryRowContext(ctx, query, args...)

	var t Tenant
	err := row.Scan(
		&t.ID,
		&t.CreatedAt,
		&t.Version,
		&t.Name,
		&t.WABAID,
		&t.PhoneNumberID,
		&t.DisplayPhoneNumber,
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
