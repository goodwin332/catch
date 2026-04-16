package domain

import (
	"errors"
	"net/mail"
	"strings"
)

type Email string

func NewEmail(value string) (Email, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return "", errors.New("email is required")
	}
	if _, err := mail.ParseAddress(normalized); err != nil {
		return "", errors.New("email is invalid")
	}
	return Email(normalized), nil
}

func (e Email) String() string {
	return string(e)
}
