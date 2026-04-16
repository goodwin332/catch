package domain

import (
	"errors"
	"strings"
	"time"
)

type SubmissionStatus string
type ThreadStatus string

const (
	SubmissionStatusPending   SubmissionStatus = "pending"
	SubmissionStatusApproved  SubmissionStatus = "approved"
	SubmissionStatusRejected  SubmissionStatus = "rejected"
	SubmissionStatusCancelled SubmissionStatus = "cancelled"

	ThreadStatusOpen     ThreadStatus = "open"
	ThreadStatusResolved ThreadStatus = "resolved"
)

var (
	ErrInvalidThread    = errors.New("invalid moderation thread")
	ErrInvalidRejection = errors.New("invalid moderation rejection")
	ErrNotFound         = errors.New("moderation item not found")
	ErrAlreadyDecided   = errors.New("moderation already decided")
)

type Submission struct {
	ID              string
	ArticleID       string
	RevisionID      string
	AuthorID        string
	Status          SubmissionStatus
	RejectionReason string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Thread struct {
	ID           string
	SubmissionID string
	AuthorID     string
	BlockID      string
	Body         string
	Status       ThreadStatus
	CreatedAt    time.Time
}

func NormalizeThread(blockID, body string) (string, string, error) {
	cleanBody := strings.TrimSpace(body)
	if cleanBody == "" {
		return "", "", ErrInvalidThread
	}
	return strings.TrimSpace(blockID), cleanBody, nil
}

func NormalizeRejection(reason string) (string, error) {
	clean := strings.TrimSpace(reason)
	if clean == "" {
		return "", ErrInvalidRejection
	}
	return clean, nil
}
