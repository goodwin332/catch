package search

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMeiliArticleIndexerIndexesAndDeletesArticle(t *testing.T) {
	var indexed []ArticleDocument
	var deletedPath string
	settingsSeen := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization header = %q, want bearer key", r.Header.Get("Authorization"))
		}
		switch {
		case r.Method == http.MethodPatch && r.URL.Path == "/indexes/catch_articles/settings":
			var settings map[string]any
			if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
				t.Fatalf("decode settings body: %v", err)
			}
			if len(settings["searchableAttributes"].([]any)) == 0 {
				t.Fatal("searchable attributes are empty")
			}
			settingsSeen = true
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodPost && r.URL.Path == "/indexes/catch_articles/documents":
			if err := json.NewDecoder(r.Body).Decode(&indexed); err != nil {
				t.Fatalf("decode index body: %v", err)
			}
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodDelete && r.URL.Path == "/indexes/catch_articles/documents/article-1":
			deletedPath = r.URL.Path
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	indexer := NewMeiliArticleIndexer(MeiliConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Index:   "catch_articles",
		Client:  server.Client(),
	})
	document := ArticleDocument{
		ID:          "article-1",
		AuthorID:    "author-1",
		Title:       "Ловля на реке",
		Excerpt:     "Короткое описание",
		Body:        "Текст статьи",
		Tags:        []string{"рыбалка"},
		PublishedAt: time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := indexer.IndexArticle(context.Background(), document); err != nil {
		t.Fatalf("index article: %v", err)
	}
	if !settingsSeen {
		t.Fatal("index settings request was not sent")
	}
	if len(indexed) != 1 || indexed[0].ID != document.ID || indexed[0].Title != document.Title {
		t.Fatalf("indexed = %+v, want one article document", indexed)
	}

	if err := indexer.DeleteArticle(context.Background(), document.ID); err != nil {
		t.Fatalf("delete article: %v", err)
	}
	if deletedPath == "" {
		t.Fatal("delete request was not sent")
	}
}

func TestMeiliArticleIndexerSearchesArticleIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/indexes/catch_articles/search" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode search request: %v", err)
		}
		if request["q"] != "щука" || request["limit"].(float64) != 2 || request["offset"].(float64) != 4 {
			t.Fatalf("search request = %+v, want q/limit/offset", request)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"hits": []map[string]any{
				{"id": "article-1"},
				{"id": "article-2"},
			},
			"estimatedTotalHits": 9,
		})
	}))
	defer server.Close()

	indexer := NewMeiliArticleIndexer(MeiliConfig{BaseURL: server.URL, Index: "catch_articles", Client: server.Client()})
	result, err := indexer.SearchArticles(context.Background(), ArticleSearchRequest{Query: "щука", Limit: 2, Offset: 4})
	if err != nil {
		t.Fatalf("search articles: %v", err)
	}
	if len(result.IDs) != 2 || result.IDs[0] != "article-1" || result.IDs[1] != "article-2" {
		t.Fatalf("ids = %v, want ordered article ids", result.IDs)
	}
	if result.NextOffset == nil || *result.NextOffset != 6 {
		t.Fatalf("next offset = %v, want 6", result.NextOffset)
	}
}
