package domain

import "time"

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const DefaultListName = "Избранное"

type List struct {
	ID       string
	UserID   string
	Name     string
	Position int
}

type Article struct {
	ListID       string
	ListName     string
	ArticleID    string
	AuthorID     string
	Title        string
	Excerpt      string
	Tags         []string
	PublishedAt  time.Time
	BookmarkedAt time.Time
}

var (
	ErrInvalidListName = errors.New("invalid bookmark list name")
	ErrLimitExceeded   = errors.New("bookmark limit exceeded")
)

func NormalizeListName(value string) (string, error) {
	name := strings.TrimSpace(value)
	length := utf8.RuneCountInString(name)
	if length < 1 || length > 80 {
		return "", ErrInvalidListName
	}
	return name, nil
}
