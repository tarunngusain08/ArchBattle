package shared

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Tier string

type Topic string

type Mode string

type ContextKey string

const (
	TierJunior Tier = "junior"
	TierSenior Tier = "senior"
	TierStaff  Tier = "staff"
)

const (
	ModeFFF Mode = "fff"
)

const (
	UserIDContextKey   ContextKey = "user_id"
	UsernameContextKey ContextKey = "username"
	TierContextKey     ContextKey = "tier"
	RoleContextKey     ContextKey = "role"
)

const (
	QuestionsPerMatch = 5
)

type QuestionSnapshot struct {
	ID             uuid.UUID `json:"id"`
	Prompt         string    `json:"prompt"`
	Options        []string  `json:"options"`
	CorrectAnswers []int     `json:"correctAnswers,omitempty"`
	Rationale      string    `json:"rationale,omitempty"`
	Topic          Topic     `json:"topic"`
	DifficultyTier Tier      `json:"difficultyTier"`
	Mode           Mode      `json:"mode"`
	Status         string    `json:"status,omitempty"` // pilot, live, etc.; not sent to clients
}

// ToClientSnapshot returns a copy of the snapshot safe to broadcast during active play.
// CorrectAnswers and Rationale are stripped and only exposed during question_reveal.
func (q *QuestionSnapshot) ToClientSnapshot() *QuestionSnapshot {
	if q == nil {
		return nil
	}
	return &QuestionSnapshot{
		ID:             q.ID,
		Prompt:         q.Prompt,
		Options:        append([]string(nil), q.Options...),
		Topic:          q.Topic,
		DifficultyTier: q.DifficultyTier,
		Mode:           q.Mode,
	}
}

func ParseTier(value string) (Tier, error) {
	normalized := Tier(strings.ToLower(strings.TrimSpace(value)))
	switch normalized {
	case TierJunior, TierSenior, TierStaff:
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported tier %q", value)
	}
}

func NormalizeTopic(value string) Topic {
	return Topic(strings.ToLower(strings.TrimSpace(value)))
}

func CurrentWeekKey(now time.Time) string {
	year, week := now.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

func ClampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
