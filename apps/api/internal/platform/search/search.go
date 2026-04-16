package search

import (
	"context"
	"errors"
	"time"
)

var ErrUnavailable = errors.New("search provider unavailable")

type ArticleDocument struct {
	ID          string    `json:"id"`
	AuthorID    string    `json:"author_id"`
	Title       string    `json:"title"`
	Excerpt     string    `json:"excerpt"`
	Body        string    `json:"body"`
	Tags        []string  `json:"tags"`
	PublishedAt time.Time `json:"published_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ArticleIndexer interface {
	IndexArticle(context.Context, ArticleDocument) error
	DeleteArticle(context.Context, string) error
}

type ArticleSearcher interface {
	SearchArticles(context.Context, ArticleSearchRequest) (ArticleSearchResult, error)
}

type ArticleSearchRequest struct {
	Query  string
	Limit  int
	Offset int
}

type ArticleSearchResult struct {
	IDs        []string
	NextOffset *int
}

type NoopArticleIndexer struct{}

func (NoopArticleIndexer) IndexArticle(context.Context, ArticleDocument) error {
	return nil
}

func (NoopArticleIndexer) DeleteArticle(context.Context, string) error {
	return nil
}

type NoopArticleSearcher struct{}

func (NoopArticleSearcher) SearchArticles(context.Context, ArticleSearchRequest) (ArticleSearchResult, error) {
	return ArticleSearchResult{}, ErrUnavailable
}
