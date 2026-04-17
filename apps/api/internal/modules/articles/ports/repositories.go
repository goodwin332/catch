package ports

import (
	"context"
	"encoding/json"
	"time"

	"catch/apps/api/internal/modules/articles/domain"
)

type Repository interface {
	CreateDraft(context.Context, CreateDraftInput) (domain.Draft, error)
	FindDraftForAuthor(context.Context, string, string) (domain.Draft, error)
	ListForAuthor(context.Context, string, int) ([]domain.Draft, error)
	FindPublished(context.Context, string, time.Time) (domain.Draft, error)
	ListPublished(context.Context, ListPublishedInput) ([]domain.Draft, error)
	ListPublishedByIDs(context.Context, ListPublishedByIDsInput) ([]domain.Draft, error)
	ListPopular(context.Context, ListPopularInput) ([]domain.Draft, error)
	ListPersonalizedFeed(context.Context, PersonalizedFeedInput) ([]domain.Draft, error)
	SearchPublished(context.Context, SearchPublishedInput) ([]domain.Draft, error)
	UpdateDraftRevision(context.Context, UpdateDraftRevisionInput) (domain.Draft, error)
	SubmitDraft(context.Context, SubmitDraftInput) (domain.Draft, error)
	ArchiveDraft(context.Context, ArchiveDraftInput) (domain.Draft, error)
	CountPublishedByAuthorSince(context.Context, CountPublishedByAuthorSinceInput) (int, error)
}

type CreateDraftInput struct {
	AuthorID string
	Title    string
	Content  json.RawMessage
	Excerpt  string
	Tags     []string
}

type UpdateDraftRevisionInput struct {
	ArticleID string
	AuthorID  string
	Title     string
	Content   json.RawMessage
	Excerpt   string
	Tags      []string
}

type SubmitDraftInput struct {
	ArticleID          string
	AuthorID           string
	RevisionStatus     domain.RevisionStatus
	ArticleStatus      domain.ArticleStatus
	ModerationRequired bool
	ScheduledAt        *time.Time
	PublishedAt        *time.Time
	RewardPublication  bool
}

type ArchiveDraftInput struct {
	ArticleID string
	AuthorID  string
}

type CountPublishedByAuthorSinceInput struct {
	AuthorID string
	Since    time.Time
}

type ListPublishedInput struct {
	Limit  int
	Now    time.Time
	Cursor *ListCursor
}

type ListPublishedByIDsInput struct {
	IDs []string
	Now time.Time
}

type ListPopularInput struct {
	Limit int
	Now   time.Time
	Since time.Time
}

type PersonalizedFeedInput struct {
	UserID string
	Limit  int
	Now    time.Time
	Cursor *ListCursor
}

type SearchPublishedInput struct {
	Query  string
	Limit  int
	Now    time.Time
	Cursor *ListCursor
}

type ListCursor struct {
	Rank        int
	PublishedAt time.Time
	ID          string
}
