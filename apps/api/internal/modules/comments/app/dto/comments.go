package dto

type CreateCommentRequest struct {
	ParentID string `json:"parent_id,omitempty"`
	Body     string `json:"body"`
}

type UpdateCommentRequest struct {
	Body string `json:"body"`
}

type CommentResponse struct {
	ID            string `json:"id"`
	ArticleID     string `json:"article_id"`
	AuthorID      string `json:"author_id"`
	ParentID      string `json:"parent_id,omitempty"`
	Body          string `json:"body"`
	Status        string `json:"status"`
	ReactionsUp   int    `json:"reactions_up"`
	ReactionsDown int    `json:"reactions_down"`
	ReactionScore int    `json:"reaction_score"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	EditedAt      string `json:"edited_at,omitempty"`
}

type CommentListResponse struct {
	Items []CommentResponse `json:"items"`
}
