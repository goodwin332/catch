package ports

import (
	"context"

	"catch/apps/api/internal/modules/comments/domain"
)

type Repository interface {
	ListByArticle(context.Context, string) ([]domain.Comment, error)
	FindByID(context.Context, string) (domain.Comment, error)
	Create(context.Context, CreateCommentInput) (domain.Comment, error)
	UpdateBody(context.Context, UpdateCommentInput) (domain.Comment, error)
}

type CreateCommentInput struct {
	ArticleID string
	AuthorID  string
	ParentID  string
	Body      string
}

type UpdateCommentInput struct {
	CommentID string
	Body      string
}
