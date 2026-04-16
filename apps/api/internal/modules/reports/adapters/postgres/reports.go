package postgres

import (
	"context"
	"errors"

	"catch/apps/api/internal/modules/reports/domain"
	"catch/apps/api/internal/modules/reports/ports"
	"catch/apps/api/internal/platform/db"
	"catch/apps/api/internal/platform/events"

	"github.com/jackc/pgx/v5"
)

type Repository struct {
	tx *db.TxManager
}

func NewRepository(tx *db.TxManager) *Repository {
	return &Repository{tx: tx}
}

func (r *Repository) Create(ctx context.Context, input ports.CreateReportInput) (domain.Report, error) {
	if err := r.ensureTargetExists(ctx, input.TargetType, input.TargetID); err != nil {
		return domain.Report{}, err
	}
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		insert into reports (target_type, target_id, reporter_id, reason, details)
		values ($1, $2, $3, $4, nullif($5, ''))
		on conflict (target_type, target_id, reporter_id, reason) do update
			set created_at = reports.created_at
		returning id::text, target_type, target_id::text, reporter_id::text, reason, coalesce(details, ''), status, created_at, decided_at
	`, input.TargetType, input.TargetID, input.ReporterID, input.Reason, input.Details)
	return scanReport(row)
}

func (r *Repository) ListPending(ctx context.Context, limit int) ([]domain.Report, error) {
	rows, err := r.tx.Querier(ctx).Query(ctx, `
		select id::text, target_type, target_id::text, reporter_id::text, reason, coalesce(details, ''), status, created_at, decided_at
		from reports
		where status = 'pending'
		order by created_at asc
		limit $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []domain.Report
	for rows.Next() {
		report, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return reports, nil
}

func (r *Repository) Decide(ctx context.Context, input ports.DecideReportInput) (domain.Report, error) {
	report, err := r.findPendingForUpdate(ctx, input.ReportID)
	if err != nil {
		return domain.Report{}, err
	}
	if report.Status != domain.StatusPending {
		return domain.Report{}, domain.ErrReportDecided
	}

	if _, err := r.tx.Querier(ctx).Exec(ctx, `
		insert into report_decisions (report_id, moderator_id, decision, is_admin_decision)
		values ($1, $2, $3, $4)
		on conflict (report_id, moderator_id) do update
			set decision = excluded.decision,
				is_admin_decision = excluded.is_admin_decision,
				created_at = now()
	`, input.ReportID, input.ModeratorID, string(input.Decision), input.IsAdminDecision); err != nil {
		return domain.Report{}, err
	}

	shouldFinalize, nextStatus, err := r.shouldFinalize(ctx, report, input.Decision, input.IsAdminDecision)
	if err != nil {
		return domain.Report{}, err
	}
	if shouldFinalize {
		if err := r.finalize(ctx, report, nextStatus); err != nil {
			return domain.Report{}, err
		}
	}
	return r.findByID(ctx, input.ReportID)
}

func (r *Repository) ensureTargetExists(ctx context.Context, targetType, targetID string) error {
	var exists bool
	var query string
	switch targetType {
	case string(domain.TargetTypeArticle):
		query = `select exists(select 1 from articles where id = $1 and status = 'published')`
	case string(domain.TargetTypeComment):
		query = `select exists(select 1 from comments where id = $1 and status = 'active')`
	default:
		return domain.ErrInvalidReport
	}
	if err := r.tx.Querier(ctx).QueryRow(ctx, query, targetID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return domain.ErrInvalidReport
	}
	return nil
}

func (r *Repository) shouldFinalize(ctx context.Context, report domain.Report, decision domain.Decision, isAdmin bool) (bool, domain.Status, error) {
	if isAdmin {
		if decision == domain.DecisionAccept {
			return true, domain.StatusAccepted, nil
		}
		return true, domain.StatusRejected, nil
	}
	var count int
	if err := r.tx.Querier(ctx).QueryRow(ctx, `
		select count(*)
		from report_decisions
		where report_id = $1 and decision = $2
	`, report.ID, string(decision)).Scan(&count); err != nil {
		return false, "", err
	}
	if count < domain.RequiredDecisions(report.TargetType, decision) {
		return false, "", nil
	}
	if decision == domain.DecisionAccept {
		return true, domain.StatusAccepted, nil
	}
	return true, domain.StatusRejected, nil
}

func (r *Repository) finalize(ctx context.Context, report domain.Report, status domain.Status) error {
	if _, err := r.tx.Querier(ctx).Exec(ctx, `
		update reports
		set status = $2, decided_at = now()
		where id = $1 and status = 'pending'
	`, report.ID, string(status)); err != nil {
		return err
	}
	if status != domain.StatusAccepted {
		return nil
	}
	if report.TargetType == domain.TargetTypeArticle {
		return r.acceptArticleReport(ctx, report)
	}
	return r.acceptCommentReport(ctx, report)
}

func (r *Repository) acceptArticleReport(ctx context.Context, report domain.Report) error {
	var authorID string
	if err := r.tx.Querier(ctx).QueryRow(ctx, `
		update articles
		set status = 'removed', removed_at = now(), updated_at = now()
		where id = $1
		returning author_id::text
	`, report.TargetID).Scan(&authorID); err != nil {
		return err
	}
	if err := r.applyPenalty(ctx, authorID, report, -100, "accepted_article_report"); err != nil {
		return err
	}
	return r.notifyReportAccepted(ctx, authorID, report)
}

func (r *Repository) acceptCommentReport(ctx context.Context, report domain.Report) error {
	var authorID string
	if err := r.tx.Querier(ctx).QueryRow(ctx, `
		update comments
		set status = 'deleted', deleted_at = now(), updated_at = now()
		where id = $1
		returning author_id::text
	`, report.TargetID).Scan(&authorID); err != nil {
		return err
	}
	if err := r.applyPenalty(ctx, authorID, report, -50, "accepted_comment_report"); err != nil {
		return err
	}
	return r.notifyReportAccepted(ctx, authorID, report)
}

func (r *Repository) notifyReportAccepted(ctx context.Context, authorID string, report domain.Report) error {
	q := r.tx.Querier(ctx)
	if err := events.Notify(ctx, q, events.NotificationInput{
		UserID:     authorID,
		EventType:  "report.accepted",
		TargetType: string(report.TargetType),
		TargetID:   report.TargetID,
		Title:      "Жалоба принята",
		Body:       "Материал скрыт после проверки жалобы.",
	}); err != nil {
		return err
	}
	return events.AddOutbox(ctx, q, "report", report.ID, "report.accepted", map[string]any{
		"report_id":   report.ID,
		"target_type": string(report.TargetType),
		"target_id":   report.TargetID,
		"author_id":   authorID,
	})
}

func (r *Repository) applyPenalty(ctx context.Context, userID string, report domain.Report, delta int, reason string) error {
	if _, err := r.tx.Querier(ctx).Exec(ctx, `
		insert into rating_events (user_id, source_type, source_id, delta, reason)
		values ($1, $2, $3, $4, $5)
	`, userID, string(report.TargetType)+"_report", report.ID, delta, reason); err != nil {
		return err
	}
	_, err := r.tx.Querier(ctx).Exec(ctx, `
		update users
		set rating = least(1000000, rating + $2), updated_at = now()
		where id = $1
	`, userID, delta)
	return err
}

func (r *Repository) findPendingForUpdate(ctx context.Context, id string) (domain.Report, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		select id::text, target_type, target_id::text, reporter_id::text, reason, coalesce(details, ''), status, created_at, decided_at
		from reports
		where id = $1
		for update
	`, id)
	return scanReport(row)
}

func (r *Repository) findByID(ctx context.Context, id string) (domain.Report, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		select id::text, target_type, target_id::text, reporter_id::text, reason, coalesce(details, ''), status, created_at, decided_at
		from reports
		where id = $1
	`, id)
	return scanReport(row)
}

type reportScanner interface {
	Scan(...any) error
}

func scanReport(row reportScanner) (domain.Report, error) {
	var report domain.Report
	var targetType, reason, status string
	if err := row.Scan(
		&report.ID,
		&targetType,
		&report.TargetID,
		&report.ReporterID,
		&reason,
		&report.Details,
		&status,
		&report.CreatedAt,
		&report.DecidedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Report{}, domain.ErrReportNotFound
		}
		return domain.Report{}, err
	}
	report.TargetType = domain.TargetType(targetType)
	report.Reason = domain.Reason(reason)
	report.Status = domain.Status(status)
	return report, nil
}
