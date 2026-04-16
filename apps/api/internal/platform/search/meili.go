package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type MeiliConfig struct {
	BaseURL string
	APIKey  string
	Index   string
	Client  *http.Client
}

type MeiliArticleIndexer struct {
	baseURL string
	apiKey  string
	index   string
	client  *http.Client
	mu      sync.Mutex
	ready   bool
}

func NewMeiliArticleIndexer(cfg MeiliConfig) *MeiliArticleIndexer {
	client := cfg.Client
	if client == nil {
		client = http.DefaultClient
	}
	return &MeiliArticleIndexer{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:  cfg.APIKey,
		index:   cfg.Index,
		client:  client,
	}
}

func (m *MeiliArticleIndexer) IndexArticle(ctx context.Context, article ArticleDocument) error {
	if err := m.EnsureArticleIndex(ctx); err != nil {
		return err
	}
	payload, err := json.Marshal([]ArticleDocument{article})
	if err != nil {
		return err
	}
	return m.do(ctx, http.MethodPost, "/indexes/"+url.PathEscape(m.index)+"/documents", bytes.NewReader(payload))
}

func (m *MeiliArticleIndexer) DeleteArticle(ctx context.Context, articleID string) error {
	if strings.TrimSpace(articleID) == "" {
		return nil
	}
	return m.do(ctx, http.MethodDelete, "/indexes/"+url.PathEscape(m.index)+"/documents/"+url.PathEscape(articleID), nil)
}

func (m *MeiliArticleIndexer) SearchArticles(ctx context.Context, request ArticleSearchRequest) (ArticleSearchResult, error) {
	if request.Limit <= 0 {
		request.Limit = 10
	}
	if request.Offset < 0 {
		request.Offset = 0
	}
	payload, err := json.Marshal(map[string]any{
		"q":      request.Query,
		"limit":  request.Limit,
		"offset": request.Offset,
	})
	if err != nil {
		return ArticleSearchResult{}, err
	}
	var response meiliSearchResponse
	if err := m.doJSON(ctx, http.MethodPost, "/indexes/"+url.PathEscape(m.index)+"/search", bytes.NewReader(payload), &response); err != nil {
		return ArticleSearchResult{}, err
	}

	ids := make([]string, 0, len(response.Hits))
	for _, hit := range response.Hits {
		if hit.ID != "" {
			ids = append(ids, hit.ID)
		}
	}
	var nextOffset *int
	if response.EstimatedTotalHits > request.Offset+len(response.Hits) {
		next := request.Offset + request.Limit
		nextOffset = &next
	}
	return ArticleSearchResult{IDs: ids, NextOffset: nextOffset}, nil
}

func (m *MeiliArticleIndexer) EnsureArticleIndex(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ready {
		return nil
	}
	payload, err := json.Marshal(map[string]any{
		"searchableAttributes": []string{"title", "tags", "excerpt", "body"},
		"displayedAttributes":  []string{"id", "author_id", "title", "excerpt", "body", "tags", "published_at", "updated_at"},
		"filterableAttributes": []string{"author_id", "tags"},
		"sortableAttributes":   []string{"published_at", "updated_at"},
		"rankingRules":         []string{"words", "typo", "proximity", "attribute", "sort", "exactness"},
	})
	if err != nil {
		return err
	}
	if err := m.do(ctx, http.MethodPatch, "/indexes/"+url.PathEscape(m.index)+"/settings", bytes.NewReader(payload)); err != nil {
		return err
	}
	m.ready = true
	return nil
}

func (m *MeiliArticleIndexer) do(ctx context.Context, method, path string, body io.Reader) error {
	return m.doJSON(ctx, method, path, body, nil)
}

func (m *MeiliArticleIndexer) doJSON(ctx context.Context, method, path string, body io.Reader, target any) error {
	request, err := http.NewRequestWithContext(ctx, method, m.baseURL+path, body)
	if err != nil {
		return err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if m.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+m.apiKey)
	}

	response, err := m.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		if target != nil {
			if err := json.NewDecoder(response.Body).Decode(target); err != nil {
				return err
			}
		}
		return nil
	}
	responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	return fmt.Errorf("meilisearch %s %s failed: status=%d body=%s", method, path, response.StatusCode, strings.TrimSpace(string(responseBody)))
}

type meiliSearchResponse struct {
	Hits []struct {
		ID string `json:"id"`
	} `json:"hits"`
	EstimatedTotalHits int `json:"estimatedTotalHits"`
}
