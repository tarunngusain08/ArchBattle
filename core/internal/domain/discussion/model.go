package discussion

import (
	"time"

	"github.com/google/uuid"
)

// Entry represents a user's discussion contribution for a daily challenge question.
type Entry struct {
	ID             uuid.UUID `json:"id"`
	ChallengeDate  time.Time `json:"challengeDate"`
	UserID         uuid.UUID `json:"userId"`
	Username       string    `json:"username"`
	QuestionNumber int       `json:"questionNumber"`
	ReasoningText  string    `json:"reasoningText"`
	AlternativeText string  `json:"alternativeText"`
	SurpriseText   string    `json:"surpriseText"`
	Upvotes        int       `json:"upvotes"`
	CreatedAt      time.Time `json:"createdAt"`
}

// CreateRequest is the input for creating a new discussion entry.
type CreateRequest struct {
	ChallengeDate  time.Time `json:"challengeDate"`
	UserID         uuid.UUID `json:"userId"`
	QuestionNumber int       `json:"questionNumber"`
	ReasoningText  string    `json:"reasoningText"`
	AlternativeText string   `json:"alternativeText"`
	SurpriseText   string    `json:"surpriseText"`
}
