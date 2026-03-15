package question

import (
	"time"

	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type Question struct {
	ID               uuid.UUID    `json:"id"`
	Mode             shared.Mode  `json:"mode"`
	Topic            shared.Topic `json:"topic"`
	DifficultyTier   shared.Tier  `json:"difficultyTier"`
	Prompt           string       `json:"prompt"`
	Options          []string     `json:"options"`
	CorrectAnswers   []int        `json:"correctAnswers"`
	Rationale        string       `json:"rationale"`
	DisputeCount     int          `json:"disputeCount"`
	PilotAttempts    int          `json:"pilotAttempts"`
	PilotDisputeRate float64      `json:"pilotDisputeRate"`
	IsActive         bool         `json:"isActive"`
	DailyEligible    bool         `json:"dailyEligible"`
	ReviewedBy       *uuid.UUID   `json:"reviewedBy,omitempty"`
	SecondReviewer   *uuid.UUID   `json:"secondReviewer,omitempty"`
	Status           string       `json:"status"`
	CreatedAt        time.Time    `json:"createdAt"`
}

func (q Question) GetCorrectAnswers() []int {
	return append([]int(nil), q.CorrectAnswers...)
}

type Dispute struct {
	QuestionID uuid.UUID `json:"questionId"`
	UserID     uuid.UUID `json:"userId"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `json:"createdAt"`
}
