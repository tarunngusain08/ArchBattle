package match

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type MatchRepository interface {
	Create(ctx context.Context, match *Match) error
	FindByID(ctx context.Context, id uuid.UUID) (*Match, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status MatchState) error
	UpdateQuestionIDs(ctx context.Context, id uuid.UUID, questionIDs []uuid.UUID) error
	AddPlayers(ctx context.Context, players []MatchPlayer) error
	GetPlayers(ctx context.Context, matchID uuid.UUID) ([]MatchPlayer, error)
	UpdatePlayerResult(ctx context.Context, matchID uuid.UUID, standing PlayerStanding) error
}

type SubmissionRepository interface {
	SaveSubmission(ctx context.Context, sub *AnswerSubmission) error
	ListByMatch(ctx context.Context, matchID uuid.UUID) ([]AnswerSubmission, error)
}

type MatchStateStore interface {
	SetMatchState(ctx context.Context, matchID uuid.UUID, state *MatchStateData) error
	GetMatchState(ctx context.Context, matchID uuid.UUID) (*MatchStateData, error)
	SetPlayerStatus(ctx context.Context, matchID, userID uuid.UUID, status string) error
	GetPlayerStatus(ctx context.Context, matchID, userID uuid.UUID) (string, error)
	AppendPlayer(ctx context.Context, matchID, userID uuid.UUID) error
	SetCurrentQuestion(ctx context.Context, matchID, questionID uuid.UUID, index int) error
	SetExpiry(ctx context.Context, matchID uuid.UUID, ttl time.Duration) error
}

type AnswerStore interface {
	RecordAnswer(ctx context.Context, matchID, qID, userID uuid.UUID, score float64) (bool, error)
	IncrementSeq(ctx context.Context, matchID, qID uuid.UUID) (int64, error)
	GetRank(ctx context.Context, matchID, qID, userID uuid.UUID) (int64, error)
	GetTotalAnswered(ctx context.Context, matchID, qID uuid.UUID) (int64, error)
	StoreLatestChoice(ctx context.Context, matchID, qID, userID uuid.UUID, choices []int) error
	GetAllLatestChoices(ctx context.Context, matchID, qID uuid.UUID) (map[uuid.UUID][]int, error)
	RecordFirstAnswerTime(ctx context.Context, matchID, qID, userID uuid.UUID, score float64) (bool, error)
	ClearQuestion(ctx context.Context, matchID, qID uuid.UUID) error
	SetQuestionTTL(ctx context.Context, matchID, qID uuid.UUID, ttl time.Duration) error
}

type EventPublisher interface {
	Publish(ctx context.Context, matchID uuid.UUID, event *MatchEvent) error
	ReadEvents(ctx context.Context, matchID uuid.UUID, fromSeq string) ([]*MatchEvent, error)
	Stream(ctx context.Context, matchID uuid.UUID, lastID string, block time.Duration, handler func(*MatchEvent) error) error
}

type Broadcaster interface {
	BroadcastToMatch(ctx context.Context, matchID uuid.UUID, event *MatchEvent) error
	SendToPlayer(ctx context.Context, userID uuid.UUID, event *MatchEvent) error
}

type QuestionProvider interface {
	SelectQuestion(ctx context.Context, seenBy []uuid.UUID, tier shared.Tier, topic shared.Topic, mode shared.Mode, exclude []uuid.UUID, excludePilot bool) (*shared.QuestionSnapshot, error)
	IncrementPilotAttempt(ctx context.Context, questionID uuid.UUID) error
}

type PlayerProgressStore interface {
	GetPlayerProfiles(ctx context.Context, tier shared.Tier, userIDs []uuid.UUID) ([]PlayerProfile, error)
	UpdatePlayerProgress(ctx context.Context, tier shared.Tier, standing PlayerStanding) error
}

type LeaderboardRecorder interface {
	// RecordELO stores the player's new absolute ELO on the leaderboard.
	RecordELO(ctx context.Context, tier shared.Tier, userID uuid.UUID, absoluteELO int, at time.Time) error
}

type SummaryRequester interface {
	RequestLearningSummary(ctx context.Context, req LearningSummaryRequest) (*LearningSummary, error)
}
