package postgres

import (
	"context"
	"errors"
	"time"

	"catch/apps/api/internal/modules/identity/domain"
	"catch/apps/api/internal/modules/identity/ports"
	"catch/apps/api/internal/platform/db"

	"github.com/jackc/pgx/v5"
)

type EmailCodeRepository struct {
	tx *db.TxManager
}

func NewEmailCodeRepository(tx *db.TxManager) *EmailCodeRepository {
	return &EmailCodeRepository{tx: tx}
}

func (r *EmailCodeRepository) Create(ctx context.Context, input ports.CreateEmailCodeInput) error {
	_, err := r.tx.Querier(ctx).Exec(ctx, `
		insert into email_login_codes (email, code_hash, purpose, request_ip, expires_at)
		values ($1, $2, $3, nullif($4, '')::inet, $5)
	`, input.Email.String(), input.CodeHash, string(input.Purpose), input.RequestIP, input.ExpiresAt)
	return err
}

func (r *EmailCodeRepository) Consume(ctx context.Context, input ports.ConsumeEmailCodeInput) (ports.EmailCodePurpose, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		update email_login_codes
		set consumed_at = $3
		where id = (
			select id
			from email_login_codes
			where lower(email) = lower($1)
				and code_hash = $2
				and consumed_at is null
				and expires_at > $3
				and attempts < 5
			order by created_at desc
			limit 1
		)
		returning purpose
	`, input.Email.String(), input.CodeHash, input.Now)

	var purpose string
	if err := row.Scan(&purpose); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", domain.ErrInvalidCode
		}
		return "", err
	}

	return ports.EmailCodePurpose(purpose), nil
}

func (r *EmailCodeRepository) IncrementAttempts(ctx context.Context, email domain.Email, now time.Time) error {
	_, err := r.tx.Querier(ctx).Exec(ctx, `
		update email_login_codes
		set attempts = attempts + 1
		where lower(email) = lower($1)
			and consumed_at is null
			and expires_at > $2
			and attempts < 5
	`, email.String(), now)
	return err
}
