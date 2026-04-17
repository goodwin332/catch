package ports

import (
	"context"

	"catch/apps/api/internal/modules/moderation/domain"
)

type Repository interface {
	ListPending(context.Context, int) ([]domain.Submission, error)
	ListThreads(context.Context, string) ([]domain.Thread, error)
	Approve(context.Context, DecisionInput) (domain.Submission, error)
	Reject(context.Context, RejectInput) (domain.Submission, error)
	CreateThread(context.Context, CreateThreadInput) (domain.Thread, error)
	ResolveThread(context.Context, ResolveThreadInput) (domain.Thread, error)
	ReopenThread(context.Context, ReopenThreadInput) (domain.Thread, error)
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

type ReopenThreadInput struct {
	ThreadID string
	ActorID  string
	Reason   string
}
