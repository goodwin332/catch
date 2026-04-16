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

type SessionRepository struct {
	tx *db.TxManager
}

func NewSessionRepository(tx *db.TxManager) *SessionRepository {
	return &SessionRepository{tx: tx}
}

func (r *SessionRepository) Create(ctx context.Context, input ports.CreateSessionInput) error {
	_, err := r.tx.Querier(ctx).Exec(ctx, `
		insert into auth_sessions (user_id, token_hash, csrf_token_hash, user_agent, ip, expires_at)
		values ($1, $2, $3, nullif($4, ''), nullif($5, '')::inet, $6)
	`, input.UserID, input.TokenHash, input.CSRFTokenHash, input.UserAgent, input.IP, input.ExpiresAt)
	return err
}

func (r *SessionRepository) FindUserByTokenHash(ctx context.Context, tokenHash []byte, now time.Time) (domain.SessionUser, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		select
			s.id::text,
			s.csrf_token_hash,
			s.expires_at,
			u.id::text,
			u.email,
			coalesce(u.username, ''),
			coalesce(u.display_name, ''),
			coalesce(u.avatar_url, ''),
			u.rating,
			u.role,
			u.status,
			u.created_at,
			u.updated_at
		from auth_sessions s
		join users u on u.id = s.user_id
		where s.token_hash = $1
			and s.revoked_at is null
			and s.expires_at > $2
			and u.status = 'active'
	`, tokenHash, now)

	var result domain.SessionUser
	var email string
	if err := row.Scan(
		&result.SessionID,
		&result.CSRFTokenHash,
		&result.ExpiresAt,
		&result.User.ID,
		&email,
		&result.User.Username,
		&result.User.DisplayName,
		&result.User.AvatarURL,
		&result.User.Rating,
		&result.User.Role,
		&result.User.Status,
		&result.User.CreatedAt,
		&result.User.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.SessionUser{}, domain.ErrSessionNotFound
		}
		return domain.SessionUser{}, err
	}

	parsedEmail, err := domain.NewEmail(email)
	if err != nil {
		return domain.SessionUser{}, err
	}
	result.User.Email = parsedEmail

	return result, nil
}

func (r *SessionRepository) RevokeByTokenHash(ctx context.Context, tokenHash []byte, now time.Time) error {
	_, err := r.tx.Querier(ctx).Exec(ctx, `
		update auth_sessions
		set revoked_at = coalesce(revoked_at, $2), updated_at = $2
		where token_hash = $1
	`, tokenHash, now)
	return err
}
