package ports

import (
	"context"

	"catch/apps/api/internal/modules/reports/domain"
)

type Repository interface {
	Create(context.Context, CreateReportInput) (domain.Report, error)
	ListPending(context.Context, int) ([]domain.Report, error)
	Decide(context.Context, DecideReportInput) (domain.Report, error)
}

type CreateReportInput struct {
	TargetType string
	TargetID   string
	ReporterID string
	Reason     string
	Details    string
}

type DecideReportInput struct {
	ReportID        string
	ModeratorID     string
	Decision        domain.Decision
	IsAdminDecision bool
}
