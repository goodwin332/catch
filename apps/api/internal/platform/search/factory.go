package search

import (
	"net/http"

	"catch/apps/api/internal/app/config"
)

func NewArticleIndexer(cfg config.SearchConfig) ArticleIndexer {
	if cfg.Provider != "meilisearch" {
		return NoopArticleIndexer{}
	}
	return NewMeiliArticleIndexer(MeiliConfig{
		BaseURL: cfg.MeiliURL,
		APIKey:  cfg.MeiliAPIKey,
		Index:   cfg.MeiliIndex,
		Client: &http.Client{
			Timeout: cfg.RequestTimeout,
		},
	})
}

func NewArticleSearcher(cfg config.SearchConfig) ArticleSearcher {
	if cfg.Provider != "meilisearch" {
		return NoopArticleSearcher{}
	}
	return NewMeiliArticleIndexer(MeiliConfig{
		BaseURL: cfg.MeiliURL,
		APIKey:  cfg.MeiliAPIKey,
		Index:   cfg.MeiliIndex,
		Client: &http.Client{
			Timeout: cfg.RequestTimeout,
		},
	})
}
