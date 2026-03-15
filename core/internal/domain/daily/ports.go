package daily

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/leaderboard"
)

type DailyChallengeRepository interface {
	GetByDate(ctx context.Context, date time.Time) (*DailyChallenge, error)
	SavePlayerResult(ctx context.Context, result Result) error
	GetPlayerResult(ctx context.Context, userID uuid.UUID, date time.Time) (*Result, error)
	GetUserStreak(ctx context.Context, userID uuid.UUID) (*Streak, error)
	UpdateUserStreak(ctx context.Context, userID uuid.UUID, streak Streak) error
	BufferDays(ctx context.Context, fromDate time.Time) (int, error)
	UpdateAISummary(ctx context.Context, date time.Time, summary string) error
	GetStreakFreezeAvailable(ctx context.Context, userID uuid.UUID) (int, error)
	ConsumeStreakFreeze(ctx context.Context, userID uuid.UUID) error
	CreditWeeklyFreeze(ctx context.Context) error
}

type DailyCacheStore interface {
	SetChallenge(ctx context.Context, challenge *DailyChallenge, ttl time.Duration) error
	GetChallenge(ctx context.Context, date time.Time) (*DailyChallenge, error)
	DeleteBoard(ctx context.Context, date time.Time) error
}

type DailyLeaderboardStore interface {
	AddScore(ctx context.Context, date time.Time, userID uuid.UUID, score int) error
	Percentile(ctx context.Context, date time.Time, userID uuid.UUID) (float64, error)
	Top(ctx context.Context, date time.Time, limit int) ([]leaderboard.Entry, error)
}
