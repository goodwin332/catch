package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalidFile  = errors.New("invalid media file")
	ErrFileTooLarge = errors.New("media file too large")
	ErrFileNotFound = errors.New("media file not found")
	ErrNoPreview    = errors.New("media preview is not available")
)

const (
	StatusReady   = "ready"
	StatusDeleted = "deleted"
)

type File struct {
	ID           string
	UploaderID   string
	StorageKey   string
	OriginalName string
	MimeType     string
	SizeBytes    int64
	Width        *int
	Height       *int
	Status       string
	CreatedAt    time.Time
}
