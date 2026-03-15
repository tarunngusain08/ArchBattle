package daily

import (
	"time"

	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type DailyChallenge struct {
	ID              uuid.UUID                 `json:"id"`
	ChallengeDate   time.Time                 `json:"challengeDate"`
	QuestionIDs     []uuid.UUID               `json:"questionIds"`
	Theme           string                    `json:"theme"`
	AISummary       string                    `json:"aiSummary"`
	SummaryReviewed bool                      `json:"summaryReviewed"`
	PublishedAt     *time.Time                `json:"publishedAt,omitempty"`
	Questions       []shared.QuestionSnapshot `json:"questions,omitempty"`
}

type Submission struct {
	UserID        uuid.UUID        `json:"userId"`
	ChallengeDate time.Time        `json:"challengeDate"`
	Answers       map[string][]int `json:"answers"`
	TotalMillis   int64            `json:"totalMillis"`
}

type Result struct {
	UserID        uuid.UUID `json:"userId"`
	ChallengeDate time.Time `json:"challengeDate"`
	Score         int       `json:"score"`
	Percentile    float64   `json:"percentile"`
	StreakDay     int       `json:"streakDay"`
	ShareCardText string    `json:"shareCardText"`
	CompletedAt   time.Time `json:"completedAt"`
}

type Streak struct {
	Current     int        `json:"current"`
	Longest     int        `json:"longest"`
	LastDate    *time.Time `json:"lastDate,omitempty"`
	FreezeCount int        `json:"freezeCount"`
}
