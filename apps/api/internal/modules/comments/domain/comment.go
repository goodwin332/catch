package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

type Comment struct {
	ID            string
	ArticleID     string
	AuthorID      string
	ParentID      string
	Body          string
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	EditedAt      *time.Time
	ReactionsUp   int
	ReactionsDown int
	ReactionScore int
}

var (
	ErrInvalidBody        = errors.New("invalid comment body")
	ErrCommentNotFound    = errors.New("comment not found")
	ErrCommentNotEditable = errors.New("comment not editable")
)

func NormalizeBody(value string) (string, error) {
	body := strings.TrimSpace(value)
	length := utf8.RuneCountInString(body)
	if length < 1 || length > 4000 {
		return "", ErrInvalidBody
	}
	return body, nil
}
