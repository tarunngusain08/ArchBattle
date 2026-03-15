package discussion

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	MinQuestionNumber = 1
	MaxQuestionNumber = 3
)

var (
	ErrInvalidQuestionNumber = errors.New("question number must be 1, 2, or 3")
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
