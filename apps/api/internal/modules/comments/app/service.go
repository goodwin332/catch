package app

import (
	"context"
	"errors"
	"time"

	accessdomain "catch/apps/api/internal/modules/access/domain"
	"catch/apps/api/internal/modules/comments/app/dto"
	"catch/apps/api/internal/modules/comments/domain"
	"catch/apps/api/internal/modules/comments/ports"
	httpx "catch/apps/api/internal/platform/http"
)

type Service struct {
	repo       ports.Repository
	now        func() time.Time
	editWindow time.Duration
}

func NewService(repo ports.Repository) *Service {
	return &Service{repo: repo, now: time.Now, editWindow: time.Hour}
}

func (s *Service) ListByArticle(ctx context.Context, articleID string) (dto.CommentListResponse, error) {
	comments, err := s.repo.ListByArticle(ctx, articleID)
	if err != nil {
		return dto.CommentListResponse{}, err
	}
	items := make([]dto.CommentResponse, 0, len(comments))
	for _, comment := range comments {
		items = append(items, mapComment(comment))
	}
	return dto.CommentListResponse{Items: items}, nil
}

func (s *Service) Get(ctx context.Context, commentID string) (dto.CommentResponse, error) {
	comment, err := s.repo.FindByID(ctx, commentID)
	if err != nil {
		return dto.CommentResponse{}, mapCommentError(err)
	}
	return mapComment(comment), nil
}

func (s *Service) Create(ctx context.Context, actor accessdomain.Principal, articleID string, request dto.CreateCommentRequest) (dto.CommentResponse, error) {
	if !actor.CanComment() {
		return dto.CommentResponse{}, httpx.Forbidden("Недостаточно рейтинга для комментариев")
	}
	body, err := domain.NormalizeBody(request.Body)
	if err != nil {
		return dto.CommentResponse{}, mapCommentError(err)
	}

	comment, err := s.repo.Create(ctx, ports.CreateCommentInput{
		ArticleID: articleID,
		AuthorID:  actor.UserID,
		ParentID:  request.ParentID,
		Body:      body,
	})
	if err != nil {
		return dto.CommentResponse{}, mapCommentError(err)
	}

	return mapComment(comment), nil
}

func (s *Service) Update(ctx context.Context, actor accessdomain.Principal, commentID string, request dto.UpdateCommentRequest) (dto.CommentResponse, error) {
	body, err := domain.NormalizeBody(request.Body)
	if err != nil {
		return dto.CommentResponse{}, mapCommentError(err)
	}

	existing, err := s.repo.FindByID(ctx, commentID)
	if err != nil {
		return dto.CommentResponse{}, mapCommentError(err)
	}
	if existing.AuthorID != actor.UserID {
		return dto.CommentResponse{}, httpx.Forbidden("Редактировать комментарий может только автор")
	}
	if existing.Status != "active" || s.now().Sub(existing.CreatedAt) > s.editWindow {
		return dto.CommentResponse{}, mapCommentError(domain.ErrCommentNotEditable)
	}

	updated, err := s.repo.UpdateBody(ctx, ports.UpdateCommentInput{CommentID: commentID, Body: body})
	if err != nil {
		return dto.CommentResponse{}, mapCommentError(err)
	}
	return mapComment(updated), nil
}

func mapComment(comment domain.Comment) dto.CommentResponse {
	editedAt := ""
	if comment.EditedAt != nil {
		editedAt = comment.EditedAt.Format(time.RFC3339)
	}
	return dto.CommentResponse{
		ID:            comment.ID,
		ArticleID:     comment.ArticleID,
		AuthorID:      comment.AuthorID,
		ParentID:      comment.ParentID,
		Body:          comment.Body,
		Status:        comment.Status,
		ReactionsUp:   comment.ReactionsUp,
		ReactionsDown: comment.ReactionsDown,
		ReactionScore: comment.ReactionScore,
		CreatedAt:     comment.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     comment.UpdatedAt.Format(time.RFC3339),
		EditedAt:      editedAt,
	}
}

func mapCommentError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidBody):
		return httpx.ValidationError("Комментарий должен быть от 1 до 4000 символов", map[string]any{"body": "invalid"})
	case errors.Is(err, domain.ErrCommentNotFound):
		return httpx.NewError(404, httpx.CodeNotFound, "Комментарий не найден")
	case errors.Is(err, domain.ErrCommentNotEditable):
		return httpx.NewError(409, httpx.CodeConflict, "Комментарий уже нельзя редактировать")
	default:
		return err
	}
}
