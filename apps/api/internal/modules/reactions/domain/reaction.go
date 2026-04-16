package domain

import "errors"

type TargetType string

const (
	TargetTypeArticle TargetType = "article"
	TargetTypeComment TargetType = "comment"
)

var ErrInvalidReaction = errors.New("invalid reaction")

func Validate(targetType TargetType, targetID string, value int) error {
	if targetType != TargetTypeArticle && targetType != TargetTypeComment {
		return ErrInvalidReaction
	}
	if targetID == "" {
		return ErrInvalidReaction
	}
	if value != -1 && value != 0 && value != 1 {
		return ErrInvalidReaction
	}
	return nil
}
