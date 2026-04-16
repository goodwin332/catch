package ports

import "context"

type Repository interface {
	SetReaction(context.Context, SetReactionInput) (int, error)
	Summary(context.Context, string, string) (ReactionSummary, error)
}

type SetReactionInput struct {
	ActorID    string
	TargetType string
	TargetID   string
	Value      int
}

type ReactionSummary struct {
	ReactionsUp   int
	ReactionsDown int
	ReactionScore int
}
