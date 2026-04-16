package dto

type CreateReportRequest struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	Reason     string `json:"reason"`
	Details    string `json:"details,omitempty"`
}

type DecideReportRequest struct {
	Decision string `json:"decision"`
}

type ReportResponse struct {
	ID         string `json:"id"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	ReporterID string `json:"reporter_id"`
	Reason     string `json:"reason"`
	Details    string `json:"details,omitempty"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
}

type ReportListResponse struct {
	Items []ReportResponse `json:"items"`
}
