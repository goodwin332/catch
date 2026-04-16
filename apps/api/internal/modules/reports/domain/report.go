package domain

import (
	"errors"
	"strings"
	"time"
)

type TargetType string
type Reason string
type Status string
type Decision string

const (
	TargetTypeArticle TargetType = "article"
	TargetTypeComment TargetType = "comment"

	ReasonAdvertising Reason = "advertising"
	ReasonProfanity   Reason = "profanity"
	ReasonInsult      Reason = "insult"
	ReasonFraud       Reason = "fraud"
	ReasonOther       Reason = "other"

	StatusPending  Status = "pending"
	StatusAccepted Status = "accepted"
	StatusRejected Status = "rejected"

	DecisionAccept Decision = "accept"
	DecisionReject Decision = "reject"
)

var (
	ErrInvalidReport  = errors.New("invalid report")
	ErrReportNotFound = errors.New("report not found")
	ErrReportDecided  = errors.New("report already decided")
)

type Report struct {
	ID         string
	TargetType TargetType
	TargetID   string
	ReporterID string
	Reason     Reason
	Details    string
	Status     Status
	CreatedAt  time.Time
	DecidedAt  *time.Time
}

func Normalize(targetType, targetID, reason, details string) (TargetType, string, Reason, string, error) {
	tt := TargetType(strings.TrimSpace(targetType))
	r := Reason(strings.TrimSpace(reason))
	cleanDetails := strings.TrimSpace(details)
	if tt != TargetTypeArticle && tt != TargetTypeComment {
		return "", "", "", "", ErrInvalidReport
	}
	if strings.TrimSpace(targetID) == "" {
		return "", "", "", "", ErrInvalidReport
	}
	switch r {
	case ReasonAdvertising, ReasonProfanity, ReasonInsult, ReasonFraud:
	case ReasonOther:
		if cleanDetails == "" {
			return "", "", "", "", ErrInvalidReport
		}
	default:
		return "", "", "", "", ErrInvalidReport
	}
	return tt, strings.TrimSpace(targetID), r, cleanDetails, nil
}

func RequiredDecisions(targetType TargetType, decision Decision) int {
	if targetType == TargetTypeComment {
		if decision == DecisionAccept {
			return 3
		}
		return 5
	}
	if decision == DecisionAccept {
		return 5
	}
	return 10
}
