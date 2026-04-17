package dto

import "encoding/json"

type CreateDraftRequest struct {
	Title   string          `json:"title"`
	Content json.RawMessage `json:"content,omitempty"`
	Tags    []string        `json:"tags,omitempty"`
}

type UpdateDraftRequest struct {
	Title   *string          `json:"title,omitempty"`
	Content *json.RawMessage `json:"content,omitempty"`
	Tags    []string         `json:"tags,omitempty"`
}

type SubmitDraftRequest struct {
	PublishAt *string `json:"publish_at,omitempty"`
}

type ArticleDraftResponse struct {
	ID                 string          `json:"id"`
	Status             string          `json:"status"`
	CurrentRevisionID  string          `json:"current_revision_id"`
	ModerationRequired bool            `json:"moderation_required"`
	Title              string          `json:"title"`
	Content            json.RawMessage `json:"content"`
	Excerpt            string          `json:"excerpt"`
	CoverURL           string          `json:"cover_url,omitempty"`
	Tags               []string        `json:"tags"`
	Version            int             `json:"version"`
	ScheduledAt        *string         `json:"scheduled_at,omitempty"`
	PublishedAt        *string         `json:"published_at,omitempty"`
}

type ArticleDraftListResponse struct {
	Items []ArticleDraftResponse `json:"items"`
}

type PublicArticleResponse struct {
	ID            string          `json:"id"`
	AuthorID      string          `json:"author_id"`
	Title         string          `json:"title"`
	Content       json.RawMessage `json:"content"`
	Excerpt       string          `json:"excerpt"`
	CoverURL      string          `json:"cover_url,omitempty"`
	Tags          []string        `json:"tags"`
	ReactionsUp   int             `json:"reactions_up"`
	ReactionsDown int             `json:"reactions_down"`
	ReactionScore int             `json:"reaction_score"`
	PublishedAt   string          `json:"published_at"`
}

type ArticleListItem struct {
	ID            string   `json:"id"`
	AuthorID      string   `json:"author_id"`
	Title         string   `json:"title"`
	Excerpt       string   `json:"excerpt"`
	CoverURL      string   `json:"cover_url,omitempty"`
	Tags          []string `json:"tags"`
	ReactionsUp   int      `json:"reactions_up"`
	ReactionsDown int      `json:"reactions_down"`
	ReactionScore int      `json:"reaction_score"`
	PublishedAt   string   `json:"published_at"`
}

type ArticleListResponse struct {
	Items      []ArticleListItem `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
}
