package app

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"catch/apps/api/internal/modules/articles/domain"
	"catch/apps/api/internal/modules/articles/ports"
	"catch/apps/api/internal/platform/search"
)

func TestSearchUsesExternalIndexAndHydratesArticlesFromRepository(t *testing.T) {
	repo := &searchRepositoryStub{
		byID: map[string]domain.Draft{
			"article-2": articleDraft("article-2", "Вторая статья"),
			"article-1": articleDraft("article-1", "Первая статья"),
		},
	}
	external := &articleSearcherStub{
		result: search.ArticleSearchResult{IDs: []string{"article-2", "article-1"}, NextOffset: intPtr(2)},
	}
	service := NewServiceWithSearch(nil, repo, external)

	response, err := service.Search(context.Background(), "щука", 2, "")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if external.query != "щука" || external.limit != 2 || external.offset != 0 {
		t.Fatalf("external search = query:%q limit:%d offset:%d", external.query, external.limit, external.offset)
	}
	if len(response.Items) != 2 || response.Items[0].ID != "article-2" || response.Items[1].ID != "article-1" {
		t.Fatalf("items = %+v, want external order", response.Items)
	}
	if response.NextCursor == "" {
		t.Fatal("next cursor is empty")
	}
}

func TestSearchFallsBackToPostgresWhenExternalIndexFails(t *testing.T) {
	repo := &searchRepositoryStub{
		publishedSearch: []domain.Draft{articleDraft("article-1", "Первая статья")},
	}
	external := &articleSearcherStub{err: search.ErrUnavailable}
	service := NewServiceWithSearch(nil, repo, external)

	response, err := service.Search(context.Background(), "щука", 10, "")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if !repo.usedFallback {
		t.Fatal("postgres fallback was not used")
	}
	if len(response.Items) != 1 || response.Items[0].ID != "article-1" {
		t.Fatalf("items = %+v, want fallback result", response.Items)
	}
}

type articleSearcherStub struct {
	query  string
	limit  int
	offset int
	result search.ArticleSearchResult
	err    error
}

func (s *articleSearcherStub) SearchArticles(_ context.Context, request search.ArticleSearchRequest) (search.ArticleSearchResult, error) {
	s.query = request.Query
	s.limit = request.Limit
	s.offset = request.Offset
	if s.err != nil {
		return search.ArticleSearchResult{}, s.err
	}
	return s.result, nil
}

type searchRepositoryStub struct {
	byID            map[string]domain.Draft
	publishedSearch []domain.Draft
	usedFallback    bool
}

func (r *searchRepositoryStub) ListPublishedByIDs(_ context.Context, input ports.ListPublishedByIDsInput) ([]domain.Draft, error) {
	items := make([]domain.Draft, 0, len(input.IDs))
	for _, id := range input.IDs {
		if article, ok := r.byID[id]; ok {
			items = append(items, article)
		}
	}
	return items, nil
}

func (r *searchRepositoryStub) SearchPublished(context.Context, ports.SearchPublishedInput) ([]domain.Draft, error) {
	r.usedFallback = true
	return r.publishedSearch, nil
}

func (*searchRepositoryStub) CreateDraft(context.Context, ports.CreateDraftInput) (domain.Draft, error) {
	panic("not implemented")
}

func (*searchRepositoryStub) FindDraftForAuthor(context.Context, string, string) (domain.Draft, error) {
	panic("not implemented")
}

func (*searchRepositoryStub) ListForAuthor(context.Context, string, int) ([]domain.Draft, error) {
	panic("not implemented")
}

func (*searchRepositoryStub) FindPublished(context.Context, string, time.Time) (domain.Draft, error) {
	panic("not implemented")
}

func (*searchRepositoryStub) ListPublished(context.Context, ports.ListPublishedInput) ([]domain.Draft, error) {
	panic("not implemented")
}

func (*searchRepositoryStub) ListPopular(context.Context, ports.ListPopularInput) ([]domain.Draft, error) {
	panic("not implemented")
}

func (*searchRepositoryStub) ListPersonalizedFeed(context.Context, ports.PersonalizedFeedInput) ([]domain.Draft, error) {
	panic("not implemented")
}

func (*searchRepositoryStub) UpdateDraftRevision(context.Context, ports.UpdateDraftRevisionInput) (domain.Draft, error) {
	panic("not implemented")
}

func (*searchRepositoryStub) SubmitDraft(context.Context, ports.SubmitDraftInput) (domain.Draft, error) {
	panic("not implemented")
}

func articleDraft(id, title string) domain.Draft {
	publishedAt := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	return domain.Draft{
		ID:          id,
		AuthorID:    "author-1",
		Status:      domain.ArticleStatusPublished,
		Title:       title,
		Content:     json.RawMessage(`{"type":"catch.article","version":1,"blocks":[]}`),
		Excerpt:     "Короткое описание",
		Tags:        []string{"рыбалка"},
		PublishedAt: &publishedAt,
	}
}

func intPtr(value int) *int {
	return &value
}
