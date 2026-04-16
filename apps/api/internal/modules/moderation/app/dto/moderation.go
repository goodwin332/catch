package dto

type SubmissionResponse struct {
	ID              string `json:"id"`
	ArticleID       string `json:"article_id"`
	RevisionID      string `json:"revision_id"`
	AuthorID        string `json:"author_id"`
	Status          string `json:"status"`
	RejectionReason string `json:"rejection_reason,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type SubmissionListResponse struct {
	Items []SubmissionResponse `json:"items"`
}

type CreateThreadRequest struct {
	BlockID string `json:"block_id,omitempty"`
	Body    string `json:"body"`
}

type ThreadResponse struct {
	ID           string `json:"id"`
	SubmissionID string `json:"submission_id"`
	AuthorID     string `json:"author_id"`
	BlockID      string `json:"block_id,omitempty"`
	Body         string `json:"body"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
}

type RejectSubmissionRequest struct {
	Reason string `json:"reason"`
}
