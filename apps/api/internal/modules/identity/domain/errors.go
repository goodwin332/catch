package domain

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrInvalidCode     = errors.New("invalid code")
	ErrSessionNotFound = errors.New("session not found")
)
