package discussion

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	MinQuestionNumber  = 1
	MaxQuestionNumber  = 3
	MaxReasoningLen   = 2000
	MaxAlternativeLen = 1000
	MaxSurpriseLen    = 1000
)

var (
	ErrInvalidQuestionNumber = errors.New("question number must be 1, 2, or 3")
	ErrTextTooLong           = errors.New("text exceeds maximum length")
)

// Service provides discussion entry operations.
type Service struct {
	repo Repository
}

// NewService creates a new discussion service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Add creates a new discussion entry after validating the request.
func (s *Service) Add(ctx context.Context, req CreateRequest) (*Entry, error) {
	if req.QuestionNumber < MinQuestionNumber || req.QuestionNumber > MaxQuestionNumber {
		return nil, ErrInvalidQuestionNumber
	}
	if len(req.ReasoningText) > MaxReasoningLen {
		return nil, ErrTextTooLong
	}
	if len(req.AlternativeText) > MaxAlternativeLen {
		return nil, ErrTextTooLong
	}
	if len(req.SurpriseText) > MaxSurpriseLen {
		return nil, ErrTextTooLong
	}
	return s.repo.Create(ctx, req)
}

// List returns discussion entries for a given date, ordered by upvotes DESC.
func (s *Service) List(ctx context.Context, date time.Time) ([]*Entry, error) {
	return s.repo.ListByDate(ctx, date)
}

// Upvote records an upvote from a voter on an entry.
func (s *Service) Upvote(ctx context.Context, entryID, voterID uuid.UUID) error {
	return s.repo.Upvote(ctx, entryID, voterID)
}
