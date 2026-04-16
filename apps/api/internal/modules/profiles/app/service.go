package app

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	identitydomain "catch/apps/api/internal/modules/identity/domain"
	"catch/apps/api/internal/modules/profiles/app/dto"
	"catch/apps/api/internal/modules/profiles/domain"
	"catch/apps/api/internal/modules/profiles/ports"
	"catch/apps/api/internal/platform/db"
	httpx "catch/apps/api/internal/platform/http"
)

type Service struct {
	tx   *db.TxManager
	repo ports.Repository
	now  func() time.Time
}

func NewService(tx *db.TxManager, repo ports.Repository) *Service {
	return &Service{tx: tx, repo: repo, now: time.Now}
}

func (s *Service) GetMyProfile(ctx context.Context, userID string) (dto.PrivateProfileResponse, error) {
	profile, err := s.repo.FindPrivateByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, identitydomain.ErrNotFound) {
			return dto.PrivateProfileResponse{}, httpx.NewError(http.StatusNotFound, httpx.CodeNotFound, "Профиль не найден")
		}
		if errors.Is(err, domain.ErrUsernameTaken) {
			return dto.PrivateProfileResponse{}, httpx.NewError(http.StatusConflict, httpx.CodeConflict, "Имя пользователя уже занято")
		}
		return dto.PrivateProfileResponse{}, err
	}
	return mapPrivateProfile(profile), nil
}

func (s *Service) GetPublicProfile(ctx context.Context, username string) (dto.PublicProfileResponse, error) {
	normalized, ok := domain.NormalizeUsername(username)
	if !ok || normalized == "" {
		return dto.PublicProfileResponse{}, httpx.NewError(http.StatusNotFound, httpx.CodeNotFound, "Профиль не найден")
	}

	profile, err := s.repo.FindPublicByUsername(ctx, normalized)
	if err != nil {
		if errors.Is(err, identitydomain.ErrNotFound) {
			return dto.PublicProfileResponse{}, httpx.NewError(http.StatusNotFound, httpx.CodeNotFound, "Профиль не найден")
		}
		return dto.PublicProfileResponse{}, err
	}

	return mapPublicProfile(profile), nil
}

func (s *Service) SearchPublicProfiles(ctx context.Context, query string, limit int) (dto.ProfileSearchResponse, error) {
	cleaned := strings.TrimSpace(strings.TrimPrefix(query, "@"))
	if len([]rune(cleaned)) < 2 {
		return dto.ProfileSearchResponse{}, httpx.ValidationError("Поиск авторов начинается с 2 символов", map[string]any{"q": "too_short"})
	}
	profiles, err := s.repo.SearchPublic(ctx, cleaned, normalizeLimit(limit))
	if err != nil {
		return dto.ProfileSearchResponse{}, err
	}
	items := make([]dto.PublicProfileResponse, 0, len(profiles))
	for _, profile := range profiles {
		items = append(items, mapPublicProfile(profile))
	}
	return dto.ProfileSearchResponse{Items: items}, nil
}

func (s *Service) UpdateMyProfile(ctx context.Context, userID string, request dto.UpdateMyProfileRequest) (dto.PrivateProfileResponse, error) {
	input, err := s.buildUpdateInput(userID, request)
	if err != nil {
		return dto.PrivateProfileResponse{}, err
	}

	var profile domain.Profile
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		updated, err := s.repo.UpdateByUserID(ctx, input)
		if err != nil {
			return err
		}
		profile = updated
		return nil
	})
	if err != nil {
		if errors.Is(err, identitydomain.ErrNotFound) {
			return dto.PrivateProfileResponse{}, httpx.NewError(http.StatusNotFound, httpx.CodeNotFound, "Профиль не найден")
		}
		return dto.PrivateProfileResponse{}, err
	}

	return mapPrivateProfile(profile), nil
}

func (s *Service) buildUpdateInput(userID string, request dto.UpdateMyProfileRequest) (ports.UpdateProfileInput, error) {
	input := ports.UpdateProfileInput{UserID: userID}

	if request.Username != nil {
		username, ok := domain.NormalizeUsername(*request.Username)
		if !ok {
			return ports.UpdateProfileInput{}, httpx.ValidationError("Имя пользователя указано некорректно", map[string]any{"username": "invalid"})
		}
		input.Username = &username
	}
	input.DisplayName = cleanStringPtr(request.DisplayName, 120)
	input.AvatarURL = cleanStringPtr(request.AvatarURL, 2048)
	input.Bio = cleanStringPtr(request.Bio, 1000)
	input.Boat = cleanStringPtr(request.Boat, 120)
	input.CountryCode = cleanStringPtr(request.CountryCode, 8)
	input.CountryName = cleanStringPtr(request.CountryName, 120)
	input.CityName = cleanStringPtr(request.CityName, 120)

	if request.BirthDate != nil {
		parsedBirthDate, err := parseBirthDate(*request.BirthDate, s.now())
		if err != nil {
			return ports.UpdateProfileInput{}, err
		}
		input.BirthDate = &parsedBirthDate
	}

	return input, nil
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 10
	}
	if limit > 50 {
		return 50
	}
	return limit
}

func cleanStringPtr(value *string, maxLength int) *string {
	if value == nil {
		return nil
	}
	cleaned := strings.TrimSpace(*value)
	if len([]rune(cleaned)) > maxLength {
		runes := []rune(cleaned)
		cleaned = string(runes[:maxLength])
	}
	return &cleaned
}

func parseBirthDate(value string, now time.Time) (*time.Time, error) {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.DateOnly, cleaned)
	if err != nil {
		return nil, httpx.ValidationError("Дата рождения указана некорректно", map[string]any{"birth_date": "invalid"})
	}
	if parsed.After(now) {
		return nil, httpx.ValidationError("Дата рождения не может быть в будущем", map[string]any{"birth_date": "future"})
	}

	return &parsed, nil
}

func mapPrivateProfile(profile domain.Profile) dto.PrivateProfileResponse {
	var birthDate *string
	if profile.BirthDate != nil {
		formatted := profile.BirthDate.Format(time.DateOnly)
		birthDate = &formatted
	}

	return dto.PrivateProfileResponse{
		UserID:      profile.UserID,
		Email:       profile.Email,
		Username:    profile.Username,
		DisplayName: profile.DisplayName,
		AvatarURL:   profile.AvatarURL,
		Rating:      profile.Rating,
		Role:        profile.Role,
		BirthDate:   birthDate,
		Bio:         profile.Bio,
		Boat:        profile.Boat,
		CountryCode: profile.CountryCode,
		CountryName: profile.CountryName,
		CityName:    profile.CityName,
	}
}

func mapPublicProfile(profile domain.Profile) dto.PublicProfileResponse {
	return dto.PublicProfileResponse{
		UserID:      profile.UserID,
		Username:    profile.Username,
		DisplayName: profile.DisplayName,
		AvatarURL:   profile.AvatarURL,
		Rating:      profile.Rating,
		Bio:         profile.Bio,
		Boat:        profile.Boat,
		CountryName: profile.CountryName,
		CityName:    profile.CityName,
	}
}
