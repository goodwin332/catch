package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	identitydomain "catch/apps/api/internal/modules/identity/domain"
	"catch/apps/api/internal/modules/profiles/domain"
	"catch/apps/api/internal/modules/profiles/ports"
	"catch/apps/api/internal/platform/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Repository struct {
	tx *db.TxManager
}

func NewRepository(tx *db.TxManager) *Repository {
	return &Repository{tx: tx}
}

func (r *Repository) FindPrivateByUserID(ctx context.Context, userID string) (domain.Profile, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, profileSelectSQL()+`
		where u.id = $1 and u.status <> 'deleted'
	`, userID)
	return scanProfile(row)
}

func (r *Repository) FindPublicByUsername(ctx context.Context, username string) (domain.Profile, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, profileSelectSQL()+`
		where lower(u.username) = lower($1) and u.status = 'active'
	`, username)
	return scanProfile(row)
}

func (r *Repository) SearchPublic(ctx context.Context, query string, limit int) ([]domain.Profile, error) {
	pattern := "%" + strings.ToLower(query) + "%"
	rows, err := r.tx.Querier(ctx).Query(ctx, profileSelectSQL()+`
		where u.status = 'active'
			and (
				lower(coalesce(u.username, '')) like $1
				or lower(coalesce(u.display_name, '')) like $1
			)
		order by
			case when lower(coalesce(u.username, '')) = lower($2) then 0 else 1 end,
			u.rating desc,
			u.created_at desc
		limit $3
	`, pattern, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := make([]domain.Profile, 0)
	for rows.Next() {
		profile, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return profiles, nil
}

func (r *Repository) UpdateByUserID(ctx context.Context, input ports.UpdateProfileInput) (domain.Profile, error) {
	usernameValue, usernameSet := stringValue(input.Username)
	displayNameValue, displayNameSet := stringValue(input.DisplayName)
	avatarURLValue, avatarURLSet := stringValue(input.AvatarURL)
	bioValue, bioSet := stringValue(input.Bio)
	boatValue, boatSet := stringValue(input.Boat)
	countryCodeValue, countryCodeSet := stringValue(input.CountryCode)
	countryNameValue, countryNameSet := stringValue(input.CountryName)
	cityNameValue, cityNameSet := stringValue(input.CityName)
	birthDateValue, birthDateSet := birthDateValue(input.BirthDate)

	row := r.tx.Querier(ctx).QueryRow(ctx, `
		with updated_user as (
			update users
			set
				username = case when $2 then nullif($3, '') else username end,
				display_name = case when $4 then nullif($5, '') else display_name end,
				avatar_url = case when $6 then nullif($7, '') else avatar_url end,
				updated_at = now()
			where id = $1 and status <> 'deleted'
			returning id, email, username, display_name, avatar_url, rating, role, created_at, updated_at
		),
		upsert_profile as (
			insert into user_profiles (
				user_id,
				birth_date,
				bio,
				boat,
				country_code,
				country_name,
				city_name
			)
			select
				id,
				case when $8 then $9::date else null end,
				case when $10 then nullif($11, '') else null end,
				case when $12 then nullif($13, '') else null end,
				case when $14 then nullif($15, '') else null end,
				case when $16 then nullif($17, '') else null end,
				case when $18 then nullif($19, '') else null end
			from updated_user
			on conflict (user_id) do update
			set
				birth_date = case when $8 then excluded.birth_date else user_profiles.birth_date end,
				bio = case when $10 then excluded.bio else user_profiles.bio end,
				boat = case when $12 then excluded.boat else user_profiles.boat end,
				country_code = case when $14 then excluded.country_code else user_profiles.country_code end,
				country_name = case when $16 then excluded.country_name else user_profiles.country_name end,
				city_name = case when $18 then excluded.city_name else user_profiles.city_name end,
				updated_at = now()
			returning birth_date, bio, boat, country_code, country_name, city_name
		)
		select
			u.id::text,
			u.email,
			coalesce(u.username, ''),
			coalesce(u.display_name, ''),
			coalesce(u.avatar_url, ''),
			u.rating,
			u.role,
			p.birth_date,
			coalesce(p.bio, ''),
			coalesce(p.boat, ''),
			coalesce(p.country_code, ''),
			coalesce(p.country_name, ''),
			coalesce(p.city_name, ''),
			u.created_at,
			u.updated_at
		from updated_user u
		left join upsert_profile p on true
	`,
		input.UserID,
		usernameSet,
		usernameValue,
		displayNameSet,
		displayNameValue,
		avatarURLSet,
		avatarURLValue,
		birthDateSet,
		birthDateValue,
		bioSet,
		bioValue,
		boatSet,
		boatValue,
		countryCodeSet,
		countryCodeValue,
		countryNameSet,
		countryNameValue,
		cityNameSet,
		cityNameValue,
	)

	profile, err := scanProfile(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_username_lower_unique" {
			return domain.Profile{}, domain.ErrUsernameTaken
		}
		return domain.Profile{}, err
	}
	return profile, nil
}

func profileSelectSQL() string {
	return `
		select
			u.id::text,
			u.email,
			coalesce(u.username, ''),
			coalesce(u.display_name, ''),
			coalesce(u.avatar_url, ''),
			u.rating,
			u.role,
			p.birth_date,
			coalesce(p.bio, ''),
			coalesce(p.boat, ''),
			coalesce(p.country_code, ''),
			coalesce(p.country_name, ''),
			coalesce(p.city_name, ''),
			u.created_at,
			u.updated_at
		from users u
		left join user_profiles p on p.user_id = u.id
	`
}

type profileScanner interface {
	Scan(...any) error
}

func scanProfile(row profileScanner) (domain.Profile, error) {
	var profile domain.Profile
	var birthDate *time.Time
	if err := row.Scan(
		&profile.UserID,
		&profile.Email,
		&profile.Username,
		&profile.DisplayName,
		&profile.AvatarURL,
		&profile.Rating,
		&profile.Role,
		&birthDate,
		&profile.Bio,
		&profile.Boat,
		&profile.CountryCode,
		&profile.CountryName,
		&profile.CityName,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Profile{}, identitydomain.ErrNotFound
		}
		return domain.Profile{}, err
	}
	profile.BirthDate = birthDate
	return profile, nil
}

func stringValue(value *string) (string, bool) {
	if value == nil {
		return "", false
	}
	return *value, true
}

func birthDateValue(value **time.Time) (*time.Time, bool) {
	if value == nil {
		return nil, false
	}
	return *value, true
}
