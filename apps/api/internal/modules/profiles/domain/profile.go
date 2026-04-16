package domain

import (
	"regexp"
	"strings"
	"time"
)

var usernamePattern = regexp.MustCompile(`^[a-z0-9_][a-z0-9_-]{2,39}$`)

type Profile struct {
	UserID      string
	Email       string
	Username    string
	DisplayName string
	AvatarURL   string
	Rating      int
	Role        string
	BirthDate   *time.Time
	Bio         string
	Boat        string
	CountryCode string
	CountryName string
	CityName    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NormalizeUsername(value string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return "", true
	}
	return normalized, usernamePattern.MatchString(normalized)
}
