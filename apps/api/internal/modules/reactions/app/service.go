package app

import (
	"context"
	"errors"

	accessdomain "catch/apps/api/internal/modules/access/domain"
	"catch/apps/api/internal/modules/reactions/app/dto"
	"catch/apps/api/internal/modules/reactions/domain"
	"catch/apps/api/internal/modules/reactions/ports"
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

func (s *Service) SetReaction(ctx context.Context, actor accessdomain.Principal, request dto.SetReactionRequest) (dto.ReactionResponse, error) {
	if actor.Rating < 0 && !actor.IsAdmin() {
		return dto.ReactionResponse{}, httpx.Forbidden("Недостаточно рейтинга для реакций")
	}

	targetType := domain.TargetType(request.TargetType)
	if err := domain.Validate(targetType, request.TargetID, request.Value); err != nil {
		return dto.ReactionResponse{}, mapReactionError(err)
	}

	var value int
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		updatedValue, err := s.repo.SetReaction(ctx, ports.SetReactionInput{
			ActorID:    actor.UserID,
			TargetType: string(targetType),
			TargetID:   request.TargetID,
			Value:      request.Value,
		})
		if err != nil {
			return err
		}
		value = updatedValue
		return nil
	})
	if err != nil {
		return dto.ReactionResponse{}, mapReactionError(err)
	}

	summary, err := s.repo.Summary(ctx, string(targetType), request.TargetID)
	if err != nil {
		return dto.ReactionResponse{}, err
	}
	return dto.ReactionResponse{
		TargetType:    string(targetType),
		TargetID:      request.TargetID,
		Value:         value,
		ReactionsUp:   summary.ReactionsUp,
		ReactionsDown: summary.ReactionsDown,
		ReactionScore: summary.ReactionScore,
	}, nil
}

func mapReactionError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidReaction):
		return httpx.ValidationError("Реакция указана некорректно", map[string]any{"reaction": "invalid"})
	default:
		return err
	}
}
