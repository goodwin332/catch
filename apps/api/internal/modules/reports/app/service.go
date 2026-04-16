package app

import (
	"context"
	"errors"
	"time"

	accessdomain "catch/apps/api/internal/modules/access/domain"
	"catch/apps/api/internal/modules/reports/app/dto"
	"catch/apps/api/internal/modules/reports/domain"
	"catch/apps/api/internal/modules/reports/ports"
	"catch/apps/api/internal/platform/db"
	httpx "catch/apps/api/internal/platform/http"
)

type Service struct {
	tx   *db.TxManager
	repo ports.Repository
}

func NewService(tx *db.TxManager, repo ports.Repository) *Service {
	return &Service{tx: tx, repo: repo}
}

func (s *Service) Create(ctx context.Context, actor accessdomain.Principal, request dto.CreateReportRequest) (dto.ReportResponse, error) {
	if !actor.CanReport() {
		return dto.ReportResponse{}, httpx.Forbidden("Недостаточно рейтинга для жалоб")
	}
	targetType, targetID, reason, details, err := domain.Normalize(request.TargetType, request.TargetID, request.Reason, request.Details)
	if err != nil {
		return dto.ReportResponse{}, mapReportError(err)
	}
	var report domain.Report
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		created, err := s.repo.Create(ctx, ports.CreateReportInput{
			TargetType: string(targetType),
			TargetID:   targetID,
			ReporterID: actor.UserID,
			Reason:     string(reason),
			Details:    details,
		})
		if err != nil {
			return err
		}
		report = created
		return nil
	})
	if err != nil {
		return dto.ReportResponse{}, mapReportError(err)
	}
	return mapReport(report), nil
}

func (s *Service) ListPending(ctx context.Context, actor accessdomain.Principal, limit int) (dto.ReportListResponse, error) {
	if !actor.CanModerate() {
		return dto.ReportListResponse{}, httpx.Forbidden("Недостаточно прав для модерации жалоб")
	}
	reports, err := s.repo.ListPending(ctx, normalizeLimit(limit))
	if err != nil {
		return dto.ReportListResponse{}, err
	}
	items := make([]dto.ReportResponse, 0, len(reports))
	for _, report := range reports {
		items = append(items, mapReport(report))
	}
	return dto.ReportListResponse{Items: items}, nil
}

func (s *Service) Decide(ctx context.Context, actor accessdomain.Principal, reportID string, request dto.DecideReportRequest) (dto.ReportResponse, error) {
	if !actor.CanModerate() {
		return dto.ReportResponse{}, httpx.Forbidden("Недостаточно прав для модерации жалоб")
	}
	decision := domain.Decision(request.Decision)
	if decision != domain.DecisionAccept && decision != domain.DecisionReject {
		return dto.ReportResponse{}, mapReportError(domain.ErrInvalidReport)
	}
	var report domain.Report
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		decided, err := s.repo.Decide(ctx, ports.DecideReportInput{
			ReportID:        reportID,
			ModeratorID:     actor.UserID,
			Decision:        decision,
			IsAdminDecision: actor.IsAdmin(),
		})
		if err != nil {
			return err
		}
		report = decided
		return nil
	})
	if err != nil {
		return dto.ReportResponse{}, mapReportError(err)
	}
	return mapReport(report), nil
}

func mapReport(report domain.Report) dto.ReportResponse {
	return dto.ReportResponse{
		ID:         report.ID,
		TargetType: string(report.TargetType),
		TargetID:   report.TargetID,
		ReporterID: report.ReporterID,
		Reason:     string(report.Reason),
		Details:    report.Details,
		Status:     string(report.Status),
		CreatedAt:  report.CreatedAt.Format(time.RFC3339),
	}
}

func mapReportError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidReport):
		return httpx.ValidationError("Жалоба указана некорректно", map[string]any{"report": "invalid"})
	case errors.Is(err, domain.ErrReportNotFound):
		return httpx.NewError(404, httpx.CodeNotFound, "Жалоба не найдена")
	case errors.Is(err, domain.ErrReportDecided):
		return httpx.NewError(409, httpx.CodeConflict, "Жалоба уже обработана")
	default:
		return err
	}
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}
