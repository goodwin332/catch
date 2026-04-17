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
		select
			id::text,
			article_id::text,
			revision_id::text,
			author_id::text,
			status,
			coalesce(rejection_reason, ''),
			(select count(*)::int from moderation_approvals ma where ma.submission_id = moderation_submissions.id),
			(select count(*)::int from moderation_threads mt where mt.submission_id = moderation_submissions.id and mt.status = 'open'),
			created_at,
			updated_at
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

func (r *Repository) ListThreads(ctx context.Context, submissionID string) ([]domain.Thread, error) {
	rows, err := r.tx.Querier(ctx).Query(ctx, `
		select id::text, submission_id::text, author_id::text, coalesce(block_id, ''), body, status, created_at
		from moderation_threads
		where submission_id = $1
		order by created_at asc, id asc
	`, submissionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	threads := make([]domain.Thread, 0)
	for rows.Next() {
		thread, err := scanThread(rows)
		if err != nil {
			return nil, err
		}
		threads = append(threads, thread)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return threads, nil
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
	if r.hasEnoughApprovals(ctx, input.SubmissionID) && !r.hasOpenThreads(ctx, input.SubmissionID) {
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
	submission, err := r.findSubmission(ctx, input.SubmissionID)
	if err != nil {
		return domain.Thread{}, err
	}
	if submission.Status != domain.SubmissionStatusPending {
		return domain.Thread{}, domain.ErrAlreadyDecided
	}
	if submission.AuthorID == input.AuthorID {
		return domain.Thread{}, domain.ErrAuthorAction
	}

	row := r.tx.Querier(ctx).QueryRow(ctx, `
		insert into moderation_threads (submission_id, author_id, block_id, body)
		values ($1, $2, nullif($3, ''), $4)
		returning id::text, submission_id::text, author_id::text, coalesce(block_id, ''), body, status, created_at
	`, input.SubmissionID, input.AuthorID, input.BlockID, input.Body)
	thread, err := scanThread(row)
	if err != nil {
		return domain.Thread{}, err
	}
	if err := events.Notify(ctx, r.tx.Querier(ctx), events.NotificationInput{
		UserID:     submission.AuthorID,
		EventType:  "moderation.thread.created",
		TargetType: "moderation_submission",
		TargetID:   submission.ID,
		Title:      "Новые замечания по статье",
		Body:       "Модератор оставил замечание по статье.",
	}); err != nil {
		return domain.Thread{}, err
	}
	return thread, nil
}

func (r *Repository) ResolveThread(ctx context.Context, input ports.ResolveThreadInput) (domain.Thread, error) {
	submission, err := r.findSubmissionByThread(ctx, input.ThreadID)
	if err != nil {
		return domain.Thread{}, err
	}
	if submission.AuthorID == input.ResolverID {
		return domain.Thread{}, domain.ErrAuthorAction
	}

	row := r.tx.Querier(ctx).QueryRow(ctx, `
		update moderation_threads
		set status = 'resolved', resolved_at = now(), resolved_by = $2
		where id = $1
			and status = 'open'
		returning id::text, submission_id::text, author_id::text, coalesce(block_id, ''), body, status, created_at
	`, input.ThreadID, input.ResolverID)
	thread, err := scanThread(row)
	if err != nil {
		return domain.Thread{}, err
	}
	if submission.Status == domain.SubmissionStatusPending && !r.hasOpenThreads(ctx, submission.ID) && r.hasEnoughApprovals(ctx, submission.ID) {
		if err := r.approveSubmission(ctx, submission); err != nil {
			return domain.Thread{}, err
		}
	}
	return thread, nil
}

func (r *Repository) ReopenThread(ctx context.Context, input ports.ReopenThreadInput) (domain.Thread, error) {
	submission, err := r.findSubmissionByThread(ctx, input.ThreadID)
	if err != nil {
		return domain.Thread{}, err
	}
	if submission.Status != domain.SubmissionStatusPending {
		return domain.Thread{}, domain.ErrAlreadyDecided
	}
	if submission.AuthorID == input.ActorID {
		return domain.Thread{}, domain.ErrAuthorAction
	}

	row := r.tx.Querier(ctx).QueryRow(ctx, `
		update moderation_threads
		set status = 'open', resolved_at = null, resolved_by = null
		where id = $1
			and status = 'resolved'
		returning id::text, submission_id::text, author_id::text, coalesce(block_id, ''), body, status, created_at
	`, input.ThreadID)
	thread, err := scanThread(row)
	if err != nil {
		return domain.Thread{}, err
	}
	if err := events.Notify(ctx, r.tx.Querier(ctx), events.NotificationInput{
		UserID:     submission.AuthorID,
		EventType:  "moderation.thread.reopened",
		TargetType: "moderation_submission",
		TargetID:   submission.ID,
		Title:      "Замечание снова открыто",
		Body:       "Модератор вернул замечание по статье в работу.",
	}); err != nil {
		return domain.Thread{}, err
	}
	return thread, nil
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

func (r *Repository) hasEnoughApprovals(ctx context.Context, submissionID string) bool {
	var hasAdminApproval bool
	_ = r.tx.Querier(ctx).QueryRow(ctx, `
		select exists(
			select 1
			from moderation_approvals
			where submission_id = $1
				and is_admin_approval = true
		)
	`, submissionID).Scan(&hasAdminApproval)
	return hasAdminApproval || r.approvalsCount(ctx, submissionID) >= 5
}

func (r *Repository) hasOpenThreads(ctx context.Context, submissionID string) bool {
	var exists bool
	_ = r.tx.Querier(ctx).QueryRow(ctx, `
		select exists(
			select 1
			from moderation_threads
			where submission_id = $1
				and status = 'open'
		)
	`, submissionID).Scan(&exists)
	return exists
}

func (r *Repository) findSubmissionForUpdate(ctx context.Context, id string) (domain.Submission, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		select
			id::text,
			article_id::text,
			revision_id::text,
			author_id::text,
			status,
			coalesce(rejection_reason, ''),
			(select count(*)::int from moderation_approvals ma where ma.submission_id = moderation_submissions.id),
			(select count(*)::int from moderation_threads mt where mt.submission_id = moderation_submissions.id and mt.status = 'open'),
			created_at,
			updated_at
		from moderation_submissions
		where id = $1
		for update
	`, id)
	return scanSubmission(row)
}

func (r *Repository) findSubmission(ctx context.Context, id string) (domain.Submission, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		select
			id::text,
			article_id::text,
			revision_id::text,
			author_id::text,
			status,
			coalesce(rejection_reason, ''),
			(select count(*)::int from moderation_approvals ma where ma.submission_id = moderation_submissions.id),
			(select count(*)::int from moderation_threads mt where mt.submission_id = moderation_submissions.id and mt.status = 'open'),
			created_at,
			updated_at
		from moderation_submissions
		where id = $1
	`, id)
	return scanSubmission(row)
}

func (r *Repository) findSubmissionByThread(ctx context.Context, threadID string) (domain.Submission, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		select
			ms.id::text,
			ms.article_id::text,
			ms.revision_id::text,
			ms.author_id::text,
			ms.status,
			coalesce(ms.rejection_reason, ''),
			(select count(*)::int from moderation_approvals ma where ma.submission_id = ms.id),
			(select count(*)::int from moderation_threads mt2 where mt2.submission_id = ms.id and mt2.status = 'open'),
			ms.created_at,
			ms.updated_at
		from moderation_threads mt
		join moderation_submissions ms on ms.id = mt.submission_id
		where mt.id = $1
	`, threadID)
	return scanSubmission(row)
}

type submissionScanner interface {
	Scan(...any) error
}

func scanSubmission(row submissionScanner) (domain.Submission, error) {
	var submission domain.Submission
	var status string
	if err := row.Scan(
		&submission.ID,
		&submission.ArticleID,
		&submission.RevisionID,
		&submission.AuthorID,
		&status,
		&submission.RejectionReason,
		&submission.ApprovalCount,
		&submission.OpenThreadCount,
		&submission.CreatedAt,
		&submission.UpdatedAt,
	); err != nil {
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
