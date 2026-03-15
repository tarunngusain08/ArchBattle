package question

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

// DraftRequest is the domain DTO for requesting an AI-drafted question.
type DraftRequest struct {
	Topic string
	Tier  string
	Mode  string
	Seed  string
}

// AIQuestionGenerator generates questions via AI. Used for hybrid question selection.
type AIQuestionGenerator interface {
	GenerateQuestion(ctx context.Context, topic, tier, mode string) (*Question, error)
}

type Repository interface {
	Create(ctx context.Context, question *Question) error
	GetByID(ctx context.Context, id uuid.UUID) (*Question, error)
	ListByStatus(ctx context.Context, status string) ([]Question, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	UpdateRationale(ctx context.Context, id uuid.UUID, rationale string) error
	SelectQuestion(ctx context.Context, seenBy []uuid.UUID, tier shared.Tier, topic shared.Topic, mode shared.Mode, since time.Time, exclude []uuid.UUID, excludePilot bool) (*Question, error)
	SelectFallbackRandom(ctx context.Context, tier shared.Tier, mode shared.Mode) (*Question, error)
	IncrementDispute(ctx context.Context, id uuid.UUID) (*Question, error)
	IncrementPilotAttempt(ctx context.Context, id uuid.UUID) (*Question, error)
	ListUpcomingDaily(ctx context.Context, fromDate time.Time, days int) ([]Question, error)
}
