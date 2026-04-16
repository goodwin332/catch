package postgres

import (
	"context"
	"errors"

	"catch/apps/api/internal/modules/moderation/domain"
	"catch/apps/api/internal/modules/moderation/ports"
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

func (r *Repository) ListPending(ctx context.Context, limit int) ([]domain.Submission, error) {
	rows, err := r.tx.Querier(ctx).Query(ctx, `
		select id::text, article_id::text, revision_id::text, author_id::text, status, coalesce(rejection_reason, ''), created_at, updated_at
		from moderation_submissions
		where status = 'pending'
		order by created_at asc
		limit $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var submissions []domain.Submission
	for rows.Next() {
		submission, err := scanSubmission(rows)
		if err != nil {
			return nil, err
		}
		submissions = append(submissions, submission)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return submissions, nil
}

func (r *Repository) Approve(ctx context.Context, input ports.DecisionInput) (domain.Submission, error) {
	submission, err := r.findSubmissionForUpdate(ctx, input.SubmissionID)
	if err != nil {
		return domain.Submission{}, err
	}
	if submission.Status != domain.SubmissionStatusPending {
		return domain.Submission{}, domain.ErrAlreadyDecided
	}
	if _, err := r.tx.Querier(ctx).Exec(ctx, `
		insert into moderation_approvals (submission_id, moderator_id, is_admin_approval)
		values ($1, $2, $3)
		on conflict do nothing
	`, input.SubmissionID, input.ModeratorID, input.IsAdminApproval); err != nil {
		return domain.Submission{}, err
	}
	if input.IsAdminApproval || r.approvalsCount(ctx, input.SubmissionID) >= 5 {
		if err := r.approveSubmission(ctx, submission); err != nil {
			return domain.Submission{}, err
		}
	}
	return r.findSubmission(ctx, input.SubmissionID)
}

func (r *Repository) Reject(ctx context.Context, input ports.RejectInput) (domain.Submission, error) {
	submission, err := r.findSubmissionForUpdate(ctx, input.SubmissionID)
	if err != nil {
		return domain.Submission{}, err
	}
	if submission.Status != domain.SubmissionStatusPending {
		return domain.Submission{}, domain.ErrAlreadyDecided
	}
	if _, err := r.tx.Querier(ctx).Exec(ctx, `
		update moderation_submissions
		set status = 'rejected', rejection_reason = $2, decided_at = now(), updated_at = now()
		where id = $1
	`, input.SubmissionID, input.Reason); err != nil {
		return domain.Submission{}, err
	}
	if _, err := r.tx.Querier(ctx).Exec(ctx, `
		update articles
		set status = 'archived', updated_at = now()
		where id = $1
	`, submission.ArticleID); err != nil {
		return domain.Submission{}, err
	}
	if err := events.Notify(ctx, r.tx.Querier(ctx), events.NotificationInput{
		UserID:     submission.AuthorID,
		EventType:  "moderation.rejected",
		TargetType: "article",
		TargetID:   submission.ArticleID,
		Title:      "Статья отклонена",
		Body:       "Модерация отклонила статью. Проверьте замечания и обновите материал.",
	}); err != nil {
		return domain.Submission{}, err
	}
	if err := events.AddOutbox(ctx, r.tx.Querier(ctx), "moderation_submission", submission.ID, "moderation.rejected", map[string]any{
		"submission_id": submission.ID,
		"article_id":    submission.ArticleID,
		"author_id":     submission.AuthorID,
	}); err != nil {
		return domain.Submission{}, err
	}
	return r.findSubmission(ctx, input.SubmissionID)
}

func (r *Repository) CreateThread(ctx context.Context, input ports.CreateThreadInput) (domain.Thread, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		insert into moderation_threads (submission_id, author_id, block_id, body)
		select $1, $2, nullif($3, ''), $4
		where exists (select 1 from moderation_submissions where id = $1 and status = 'pending')
		returning id::text, submission_id::text, author_id::text, coalesce(block_id, ''), body, status, created_at
	`, input.SubmissionID, input.AuthorID, input.BlockID, input.Body)
	return scanThread(row)
}

func (r *Repository) ResolveThread(ctx context.Context, input ports.ResolveThreadInput) (domain.Thread, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		update moderation_threads
		set status = 'resolved', resolved_at = now(), resolved_by = $2
		where id = $1 and status = 'open'
		returning id::text, submission_id::text, author_id::text, coalesce(block_id, ''), body, status, created_at
	`, input.ThreadID, input.ResolverID)
	return scanThread(row)
}

func (r *Repository) approveSubmission(ctx context.Context, submission domain.Submission) error {
	if _, err := r.tx.Querier(ctx).Exec(ctx, `
		update moderation_submissions
		set status = 'approved', decided_at = now(), updated_at = now()
		where id = $1
	`, submission.ID); err != nil {
		return err
	}
	if _, err := r.tx.Querier(ctx).Exec(ctx, `
		update articles
		set status = 'ready_to_publish', updated_at = now()
		where id = $1
	`, submission.ArticleID); err != nil {
		return err
	}
	if _, err := r.tx.Querier(ctx).Exec(ctx, `
		update article_revisions
		set status = 'approved'
		where id = $1
	`, submission.RevisionID); err != nil {
		return err
	}
	if err := events.Notify(ctx, r.tx.Querier(ctx), events.NotificationInput{
		UserID:     submission.AuthorID,
		EventType:  "moderation.approved",
		TargetType: "article",
		TargetID:   submission.ArticleID,
		Title:      "Статья одобрена",
		Body:       "Модерация одобрила статью. Теперь её можно опубликовать.",
	}); err != nil {
		return err
	}
	return events.AddOutbox(ctx, r.tx.Querier(ctx), "moderation_submission", submission.ID, "moderation.approved", map[string]any{
		"submission_id": submission.ID,
		"article_id":    submission.ArticleID,
		"author_id":     submission.AuthorID,
	})
}

func (r *Repository) approvalsCount(ctx context.Context, submissionID string) int {
	var count int
	_ = r.tx.Querier(ctx).QueryRow(ctx, `
		select count(*) from moderation_approvals where submission_id = $1
	`, submissionID).Scan(&count)
	return count
}

func (r *Repository) findSubmissionForUpdate(ctx context.Context, id string) (domain.Submission, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		select id::text, article_id::text, revision_id::text, author_id::text, status, coalesce(rejection_reason, ''), created_at, updated_at
		from moderation_submissions
		where id = $1
		for update
	`, id)
	return scanSubmission(row)
}

func (r *Repository) findSubmission(ctx context.Context, id string) (domain.Submission, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		select id::text, article_id::text, revision_id::text, author_id::text, status, coalesce(rejection_reason, ''), created_at, updated_at
		from moderation_submissions
		where id = $1
	`, id)
	return scanSubmission(row)
}

type submissionScanner interface {
	Scan(...any) error
}

func scanSubmission(row submissionScanner) (domain.Submission, error) {
	var submission domain.Submission
	var status string
	if err := row.Scan(&submission.ID, &submission.ArticleID, &submission.RevisionID, &submission.AuthorID, &status, &submission.RejectionReason, &submission.CreatedAt, &submission.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Submission{}, domain.ErrNotFound
		}
		return domain.Submission{}, err
	}
	submission.Status = domain.SubmissionStatus(status)
	return submission, nil
}

type threadScanner interface {
	Scan(...any) error
}

func scanThread(row threadScanner) (domain.Thread, error) {
	var thread domain.Thread
	var status string
	if err := row.Scan(&thread.ID, &thread.SubmissionID, &thread.AuthorID, &thread.BlockID, &thread.Body, &status, &thread.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Thread{}, domain.ErrNotFound
		}
		return domain.Thread{}, err
	}
	thread.Status = domain.ThreadStatus(status)
	return thread, nil
}
