package ports

import (
	"context"

	"catch/apps/api/internal/modules/bookmarks/domain"
)

type Repository interface {
	ListBookmarkLists(context.Context, string) ([]domain.List, error)
	ListBookmarkedArticles(context.Context, ListBookmarkedArticlesInput) ([]domain.Article, error)
	CreateBookmarkList(context.Context, string, string) (domain.List, error)
	EnsureDefaultList(context.Context, string) (domain.List, error)
	AddBookmark(context.Context, string, string, string) error
	RemoveBookmark(context.Context, string, string, string) error
	Follow(context.Context, string, string) (bool, error)
	Unfollow(context.Context, string, string) (bool, error)
}

type ListBookmarkedArticlesInput struct {
	UserID string
	ListID string
	Query  string
	Limit  int
}
