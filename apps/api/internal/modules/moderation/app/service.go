package app

import (
	"context"
	"errors"
	"time"

	accessdomain "catch/apps/api/internal/modules/access/domain"
	"catch/apps/api/internal/modules/moderation/app/dto"
	"catch/apps/api/internal/modules/moderation/domain"
	"catch/apps/api/internal/modules/moderation/ports"
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

func (s *Service) ListPending(ctx context.Context, actor accessdomain.Principal, limit int) (dto.SubmissionListResponse, error) {
	if !actor.CanModerate() {
		return dto.SubmissionListResponse{}, httpx.Forbidden("Недостаточно прав для модерации")
	}
	submissions, err := s.repo.ListPending(ctx, normalizeLimit(limit))
	if err != nil {
		return dto.SubmissionListResponse{}, err
	}
	items := make([]dto.SubmissionResponse, 0, len(submissions))
	for _, submission := range submissions {
		items = append(items, mapSubmission(submission))
	}
	return dto.SubmissionListResponse{Items: items}, nil
}

func (s *Service) Approve(ctx context.Context, actor accessdomain.Principal, submissionID string) (dto.SubmissionResponse, error) {
	if !actor.CanModerate() {
		return dto.SubmissionResponse{}, httpx.Forbidden("Недостаточно прав для модерации")
	}
	var submission domain.Submission
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		approved, err := s.repo.Approve(ctx, ports.DecisionInput{
			SubmissionID:    submissionID,
			ModeratorID:     actor.UserID,
			IsAdminApproval: actor.IsAdmin(),
		})
		if err != nil {
			return err
		}
		submission = approved
		return nil
	})
	if err != nil {
		return dto.SubmissionResponse{}, mapModerationError(err)
	}
	return mapSubmission(submission), nil
}

func (s *Service) ListThreads(ctx context.Context, actor accessdomain.Principal, submissionID string) (dto.ThreadListResponse, error) {
	if !actor.CanModerate() {
		return dto.ThreadListResponse{}, httpx.Forbidden("Недостаточно прав для модерации")
	}
	threads, err := s.repo.ListThreads(ctx, submissionID)
	if err != nil {
		return dto.ThreadListResponse{}, mapModerationError(err)
	}
	items := make([]dto.ThreadResponse, 0, len(threads))
	for _, thread := range threads {
		items = append(items, mapThread(thread))
	}
	return dto.ThreadListResponse{Items: items}, nil
}

func (s *Service) Reject(ctx context.Context, actor accessdomain.Principal, submissionID string, request dto.RejectSubmissionRequest) (dto.SubmissionResponse, error) {
	if !actor.IsAdmin() {
		return dto.SubmissionResponse{}, httpx.Forbidden("Отклонить статью может только администратор")
	}
	reason, err := domain.NormalizeRejection(request.Reason)
	if err != nil {
		return dto.SubmissionResponse{}, mapModerationError(err)
	}
	var submission domain.Submission
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		rejected, err := s.repo.Reject(ctx, ports.RejectInput{SubmissionID: submissionID, AdminID: actor.UserID, Reason: reason})
		if err != nil {
			return err
		}
		submission = rejected
		return nil
	})
	if err != nil {
		return dto.SubmissionResponse{}, mapModerationError(err)
	}
	return mapSubmission(submission), nil
}

func (s *Service) CreateThread(ctx context.Context, actor accessdomain.Principal, submissionID string, request dto.CreateThreadRequest) (dto.ThreadResponse, error) {
	if !actor.CanModerate() {
		return dto.ThreadResponse{}, httpx.Forbidden("Недостаточно прав для модерации")
	}
	blockID, body, err := domain.NormalizeThread(request.BlockID, request.Body)
	if err != nil {
		return dto.ThreadResponse{}, mapModerationError(err)
	}
	thread, err := s.repo.CreateThread(ctx, ports.CreateThreadInput{SubmissionID: submissionID, AuthorID: actor.UserID, BlockID: blockID, Body: body})
	if err != nil {
		return dto.ThreadResponse{}, mapModerationError(err)
	}
	return mapThread(thread), nil
}

func (s *Service) ResolveThread(ctx context.Context, actor accessdomain.Principal, threadID string) (dto.ThreadResponse, error) {
	if !actor.CanModerate() {
		return dto.ThreadResponse{}, httpx.Forbidden("Недостаточно прав для модерации")
	}
	thread, err := s.repo.ResolveThread(ctx, ports.ResolveThreadInput{ThreadID: threadID, ResolverID: actor.UserID})
	if err != nil {
		return dto.ThreadResponse{}, mapModerationError(err)
	}
	return mapThread(thread), nil
}

func (s *Service) ReopenThread(ctx context.Context, actor accessdomain.Principal, threadID string, request dto.ReopenThreadRequest) (dto.ThreadResponse, error) {
	if !actor.CanModerate() {
		return dto.ThreadResponse{}, httpx.Forbidden("Недостаточно прав для модерации")
	}
	thread, err := s.repo.ReopenThread(ctx, ports.ReopenThreadInput{ThreadID: threadID, ActorID: actor.UserID, Reason: request.Reason})
	if err != nil {
		return dto.ThreadResponse{}, mapModerationError(err)
	}
	return mapThread(thread), nil
}

func mapSubmission(submission domain.Submission) dto.SubmissionResponse {
	return dto.SubmissionResponse{
		ID:              submission.ID,
		ArticleID:       submission.ArticleID,
		RevisionID:      submission.RevisionID,
		AuthorID:        submission.AuthorID,
		Status:          string(submission.Status),
		RejectionReason: submission.RejectionReason,
		ApprovalCount:   submission.ApprovalCount,
		OpenThreadCount: submission.OpenThreadCount,
		CreatedAt:       submission.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       submission.UpdatedAt.Format(time.RFC3339),
	}
}

func mapThread(thread domain.Thread) dto.ThreadResponse {
	return dto.ThreadResponse{
		ID:           thread.ID,
		SubmissionID: thread.SubmissionID,
		AuthorID:     thread.AuthorID,
		BlockID:      thread.BlockID,
		Body:         thread.Body,
		Status:       string(thread.Status),
		CreatedAt:    thread.CreatedAt.Format(time.RFC3339),
	}
}

func mapModerationError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidThread):
		return httpx.ValidationError("Комментарий модерации не может быть пустым", map[string]any{"body": "invalid"})
	case errors.Is(err, domain.ErrInvalidRejection):
		return httpx.ValidationError("Причина отклонения обязательна", map[string]any{"reason": "required"})
	case errors.Is(err, domain.ErrNotFound):
		return httpx.NewError(404, httpx.CodeNotFound, "Объект модерации не найден")
	case errors.Is(err, domain.ErrAlreadyDecided):
		return httpx.NewError(409, httpx.CodeConflict, "Модерация уже завершена")
	case errors.Is(err, domain.ErrOpenThreads):
		return httpx.NewError(409, httpx.CodeConflict, "Нельзя завершить модерацию, пока открыты треды")
	case errors.Is(err, domain.ErrAuthorAction):
		return httpx.Forbidden("Автор статьи не может выполнять это действие модерации")
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
