package ports

import (
	"context"
	"time"

	"catch/apps/api/internal/modules/media/domain"
)

type Repository interface {
	Create(context.Context, CreateFileInput) (domain.File, error)
	FindReady(context.Context, string) (domain.File, error)
	ListUnreferencedReady(context.Context, CleanupCandidatesInput) ([]domain.File, error)
	MarkDeleted(context.Context, string) error
}

type CreateFileInput struct {
	UploaderID   string
	StorageKey   string
	OriginalName string
	MimeType     string
	SizeBytes    int64
	Width        *int
	Height       *int
}

type CleanupCandidatesInput struct {
	Before time.Time
	Limit  int
}
