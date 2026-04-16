package ports

import (
	"context"

	"catch/apps/api/internal/modules/moderation/domain"
)

type Repository interface {
	ListPending(context.Context, int) ([]domain.Submission, error)
	Approve(context.Context, DecisionInput) (domain.Submission, error)
	Reject(context.Context, RejectInput) (domain.Submission, error)
	CreateThread(context.Context, CreateThreadInput) (domain.Thread, error)
	ResolveThread(context.Context, ResolveThreadInput) (domain.Thread, error)
}

type DecisionInput struct {
	SubmissionID    string
	ModeratorID     string
	IsAdminApproval bool
}

type RejectInput struct {
	SubmissionID string
	AdminID      string
	Reason       string
}

type CreateThreadInput struct {
	SubmissionID string
	AuthorID     string
	BlockID      string
	Body         string
}

type ResolveThreadInput struct {
	ThreadID   string
	ResolverID string
}
