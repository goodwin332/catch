package dto

type FileResponse struct {
	ID           string `json:"id"`
	OriginalName string `json:"original_name"`
	MimeType     string `json:"mime_type"`
	SizeBytes    int64  `json:"size_bytes"`
	Width        *int   `json:"width,omitempty"`
	Height       *int   `json:"height,omitempty"`
	URL          string `json:"url"`
	CreatedAt    string `json:"created_at"`
}
