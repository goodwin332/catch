package app

import (
	"context"
	"errors"
	"strings"
	"time"

	accessdomain "catch/apps/api/internal/modules/access/domain"
	"catch/apps/api/internal/modules/bookmarks/app/dto"
	"catch/apps/api/internal/modules/bookmarks/domain"
	"catch/apps/api/internal/modules/bookmarks/ports"
	"catch/apps/api/internal/platform/db"
	httpx "catch/apps/api/internal/platform/http"
)

type Service struct {
	tx   *db.TxManager
	repo ports.Repository
}

func NewService(tx *db.TxManager, repo ports.Repository) *Service {
	return &Service{tx: tx, repo: repo}
}

func (s *Service) Lists(ctx context.Context, actor accessdomain.Principal) (dto.BookmarkListsResponse, error) {
	var lists []domain.List
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := s.repo.EnsureDefaultList(ctx, actor.UserID); err != nil {
			return err
		}
		loaded, err := s.repo.ListBookmarkLists(ctx, actor.UserID)
		if err != nil {
			return err
		}
		lists = loaded
		return nil
	})
	if err != nil {
		return dto.BookmarkListsResponse{}, err
	}

	items := make([]dto.BookmarkListResponse, 0, len(lists))
	for _, list := range lists {
		items = append(items, dto.BookmarkListResponse{ID: list.ID, Name: list.Name, Position: list.Position})
	}
	return dto.BookmarkListsResponse{Items: items}, nil
}

func (s *Service) CreateList(ctx context.Context, actor accessdomain.Principal, request dto.CreateBookmarkListRequest) (dto.BookmarkListResponse, error) {
	name, err := domain.NormalizeListName(request.Name)
	if err != nil {
		return dto.BookmarkListResponse{}, mapBookmarkError(err)
	}

	var list domain.List
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		created, err := s.repo.CreateBookmarkList(ctx, actor.UserID, name)
		if err != nil {
			return err
		}
		list = created
		return nil
	})
	if err != nil {
		return dto.BookmarkListResponse{}, mapBookmarkError(err)
	}

	return dto.BookmarkListResponse{ID: list.ID, Name: list.Name, Position: list.Position}, nil
}

func (s *Service) Articles(ctx context.Context, actor accessdomain.Principal, listID, query string, limit int) (dto.BookmarkedArticlesResponse, error) {
	articles, err := s.repo.ListBookmarkedArticles(ctx, ports.ListBookmarkedArticlesInput{
		UserID: actor.UserID,
		ListID: strings.TrimSpace(listID),
		Query:  strings.TrimSpace(query),
		Limit:  normalizeLimit(limit),
	})
	if err != nil {
		return dto.BookmarkedArticlesResponse{}, err
	}
	items := make([]dto.BookmarkedArticleResponse, 0, len(articles))
	for _, article := range articles {
		items = append(items, dto.BookmarkedArticleResponse{
			ListID:       article.ListID,
			ListName:     article.ListName,
			ArticleID:    article.ArticleID,
			AuthorID:     article.AuthorID,
			Title:        article.Title,
			Excerpt:      article.Excerpt,
			Tags:         article.Tags,
			PublishedAt:  article.PublishedAt.Format(time.RFC3339),
			BookmarkedAt: article.BookmarkedAt.Format(time.RFC3339),
		})
	}
	return dto.BookmarkedArticlesResponse{Items: items}, nil
}

func (s *Service) AddBookmark(ctx context.Context, actor accessdomain.Principal, request dto.AddBookmarkRequest) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		listID := request.ListID
		if listID == "" {
			list, err := s.repo.EnsureDefaultList(ctx, actor.UserID)
			if err != nil {
				return err
			}
			listID = list.ID
		}
		return s.repo.AddBookmark(ctx, actor.UserID, listID, request.ArticleID)
	})
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func (s *Service) RemoveBookmark(ctx context.Context, actor accessdomain.Principal, request dto.RemoveBookmarkRequest) error {
	return s.repo.RemoveBookmark(ctx, actor.UserID, request.ListID, request.ArticleID)
}

func (s *Service) Follow(ctx context.Context, actor accessdomain.Principal, authorID string) (dto.FollowResponse, error) {
	if !actor.CanComment() {
		return dto.FollowResponse{}, httpx.Forbidden("Недостаточно рейтинга для подписок")
	}

	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		_, err := s.repo.Follow(ctx, actor.UserID, authorID)
		return err
	})
	if err != nil {
		return dto.FollowResponse{}, err
	}
	return dto.FollowResponse{AuthorID: authorID, Following: true}, nil
}

func (s *Service) Unfollow(ctx context.Context, actor accessdomain.Principal, authorID string) (dto.FollowResponse, error) {
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		_, err := s.repo.Unfollow(ctx, actor.UserID, authorID)
		return err
	})
	if err != nil {
		return dto.FollowResponse{}, err
	}
	return dto.FollowResponse{AuthorID: authorID, Following: false}, nil
}

func mapBookmarkError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidListName):
		return httpx.ValidationError("Название списка должно быть от 1 до 80 символов", map[string]any{"name": "invalid"})
	case errors.Is(err, domain.ErrLimitExceeded):
		return httpx.ValidationError("Лимит закладок превышен", map[string]any{"limit": "exceeded"})
	default:
		return err
	}
}
