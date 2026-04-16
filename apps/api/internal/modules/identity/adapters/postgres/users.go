package postgres

import (
	"context"
	"errors"
	"strings"

	"catch/apps/api/internal/modules/identity/domain"
	"catch/apps/api/internal/modules/identity/ports"
	"catch/apps/api/internal/platform/db"

	"github.com/jackc/pgx/v5"
)

type UserRepository struct {
	tx *db.TxManager
}

func NewUserRepository(tx *db.TxManager) *UserRepository {
	return &UserRepository{tx: tx}
}

func (r *UserRepository) FindByEmail(ctx context.Context, email domain.Email) (domain.User, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		select id::text, email, coalesce(username, ''), coalesce(display_name, ''), coalesce(avatar_url, ''), rating, role, status, created_at, updated_at
		from users
		where lower(email) = lower($1) and status <> 'deleted'
	`, email.String())

	return scanUser(row)
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (domain.User, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		select id::text, email, coalesce(username, ''), coalesce(display_name, ''), coalesce(avatar_url, ''), rating, role, status, created_at, updated_at
		from users
		where id = $1 and status <> 'deleted'
	`, id)

	return scanUser(row)
}

func (r *UserRepository) FindByOAuthAccount(ctx context.Context, provider, providerAccountID string) (domain.User, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		select u.id::text, u.email, coalesce(u.username, ''), coalesce(u.display_name, ''), coalesce(u.avatar_url, ''), u.rating, u.role, u.status, u.created_at, u.updated_at
		from oauth_accounts oa
		join users u on u.id = oa.user_id
		where oa.provider = $1 and oa.provider_account_id = $2 and u.status <> 'deleted'
	`, provider, providerAccountID)

	return scanUser(row)
}

func (r *UserRepository) CreateEmailUser(ctx context.Context, input ports.CreateEmailUserInput) (domain.User, error) {
	displayName := strings.TrimSpace(input.DisplayName)
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		insert into users (email, display_name)
		values ($1, nullif($2, ''))
		on conflict ((lower(email))) do update
			set updated_at = users.updated_at
		returning id::text, email, coalesce(username, ''), coalesce(display_name, ''), coalesce(avatar_url, ''), rating, role, status, created_at, updated_at
	`, input.Email.String(), displayName)

	return scanUser(row)
}

func (r *UserRepository) LinkOAuthAccount(ctx context.Context, input ports.LinkOAuthAccountInput) error {
	_, err := r.tx.Querier(ctx).Exec(ctx, `
		insert into oauth_accounts (user_id, provider, provider_account_id, email)
		values ($1, $2, $3, $4)
		on conflict (provider, provider_account_id) do update
			set email = excluded.email,
				updated_at = now()
	`, input.UserID, input.Provider, input.ProviderAccountID, input.Email.String())
	return err
}

func (r *UserRepository) EnsureDevUser(ctx context.Context, email domain.Email) (domain.User, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		insert into users (email, display_name)
		values ($1, 'Dev User')
		on conflict ((lower(email))) do update
			set updated_at = now()
		returning id::text, email, coalesce(username, ''), coalesce(display_name, ''), coalesce(avatar_url, ''), rating, role, status, created_at, updated_at
	`, email.String())

	return scanUser(row)
}

type userScanner interface {
	Scan(...any) error
}

func scanUser(row userScanner) (domain.User, error) {
	var user domain.User
	var email string
	if err := row.Scan(
		&user.ID,
		&email,
		&user.Username,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Rating,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, err
	}

	parsedEmail, err := domain.NewEmail(email)
	if err != nil {
		return domain.User{}, err
	}
	user.Email = parsedEmail

	return user, nil
}
