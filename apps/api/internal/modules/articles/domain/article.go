package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

type ArticleStatus string

const (
	ArticleStatusDraft          ArticleStatus = "draft"
	ArticleStatusInModeration   ArticleStatus = "in_moderation"
	ArticleStatusReadyToPublish ArticleStatus = "ready_to_publish"
	ArticleStatusPublished      ArticleStatus = "published"
	ArticleStatusArchived       ArticleStatus = "archived"
	ArticleStatusRemoved        ArticleStatus = "removed"
)

type RevisionStatus string

const (
	RevisionStatusDraft     RevisionStatus = "draft"
	RevisionStatusSubmitted RevisionStatus = "submitted"
	RevisionStatusApproved  RevisionStatus = "approved"
	RevisionStatusRejected  RevisionStatus = "rejected"
	RevisionStatusPublished RevisionStatus = "published"
)

type Draft struct {
	ID                  string
	AuthorID            string
	Status              ArticleStatus
	CurrentRevisionID   string
	PublishedRevisionID string
	ModerationRequired  bool
	Title               string
	Content             json.RawMessage
	Excerpt             string
	Tags                []string
	Version             int
	ScheduledAt         *time.Time
	PublishedAt         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
	ReactionsUp         int
	ReactionsDown       int
	ReactionScore       int
	SortRank            int
}

var (
	ErrInvalidTitle       = errors.New("invalid title")
	ErrInvalidDocument    = errors.New("invalid article document")
	ErrTooManyTags        = errors.New("too many tags")
	ErrInvalidTag         = errors.New("invalid tag")
	ErrArticleNotFound    = errors.New("article not found")
	ErrArticleNotEditable = errors.New("article not editable")
	ErrPublishWindow      = errors.New("invalid publish window")
	ErrDailyPublishLimit  = errors.New("daily publish limit reached")
)

func NormalizeTitle(value string) (string, error) {
	title := strings.TrimSpace(value)
	length := utf8.RuneCountInString(title)
	if length < 3 || length > 160 {
		return "", ErrInvalidTitle
	}
	return title, nil
}

func ValidateDocument(value json.RawMessage) (json.RawMessage, error) {
	if len(value) == 0 {
		return json.RawMessage(`{"type":"catch.article","version":1,"blocks":[]}`), nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return nil, ErrInvalidDocument
	}
	if decoded == nil {
		return nil, ErrInvalidDocument
	}
	if !validateArticleBlocks(decoded) {
		return nil, ErrInvalidDocument
	}

	return value, nil
}

func validateArticleBlocks(document map[string]any) bool {
	blocks, ok := document["blocks"].([]any)
	if !ok {
		return false
	}
	for _, rawBlock := range blocks {
		block, ok := rawBlock.(map[string]any)
		if !ok {
			continue
		}
		if block["type"] != "geo_point" {
			continue
		}
		if radius, ok := block["radius_meters"].(float64); ok && (radius <= 0 || radius > 10000) {
			return false
		}
	}
	return true
}

func ExtractMediaFileIDs(value json.RawMessage) ([]string, error) {
	if len(value) == 0 {
		return nil, nil
	}
	var decoded any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return nil, ErrInvalidDocument
	}
	seen := make(map[string]bool)
	ids := make([]string, 0)
	collectMediaFileIDs(decoded, seen, &ids)
	return ids, nil
}

func collectMediaFileIDs(value any, seen map[string]bool, ids *[]string) {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if key == "file_id" || key == "media_file_id" {
				if id, ok := nested.(string); ok {
					clean := strings.TrimSpace(id)
					if clean != "" && !seen[clean] {
						seen[clean] = true
						*ids = append(*ids, clean)
					}
				}
				continue
			}
			collectMediaFileIDs(nested, seen, ids)
		}
	case []any:
		for _, nested := range typed {
			collectMediaFileIDs(nested, seen, ids)
		}
	}
}

func NormalizeTags(values []string) ([]string, error) {
	if len(values) > 10 {
		return nil, ErrTooManyTags
	}

	seen := make(map[string]bool, len(values))
	tags := make([]string, 0, len(values))
	for _, value := range values {
		tag := strings.TrimSpace(strings.TrimPrefix(value, "#"))
		if tag == "" {
			continue
		}
		if utf8.RuneCountInString(tag) > 64 {
			return nil, ErrInvalidTag
		}

		key := strings.ToLower(tag)
		if seen[key] {
			continue
		}
		seen[key] = true
		tags = append(tags, tag)
	}

	return tags, nil
}
