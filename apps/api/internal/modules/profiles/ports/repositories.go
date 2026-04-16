package ports

import (
	"context"
	"time"

	"catch/apps/api/internal/modules/profiles/domain"
)

type Repository interface {
	FindPrivateByUserID(context.Context, string) (domain.Profile, error)
	FindPublicByUsername(context.Context, string) (domain.Profile, error)
	SearchPublic(context.Context, string, int) ([]domain.Profile, error)
	UpdateByUserID(context.Context, UpdateProfileInput) (domain.Profile, error)
}

type UpdateProfileInput struct {
	UserID      string
	Username    *string
	DisplayName *string
	AvatarURL   *string
	BirthDate   **time.Time
	Bio         *string
	Boat        *string
	CountryCode *string
	CountryName *string
	CityName    *string
}
