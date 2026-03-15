package matchmaking

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type QueueStore interface {
	Enqueue(ctx context.Context, entry QueueEntry) error
	Dequeue(ctx context.Context, tier string, topic string, userID uuid.UUID) error
	FindCandidates(ctx context.Context, tier string, topic string, minELO, maxELO int) ([]QueueEntry, error)
	GetEntry(ctx context.Context, userID uuid.UUID) (*QueueEntry, error)
	GetEntryTime(ctx context.Context, userID uuid.UUID) (time.Time, error)
	GetAllActiveQueues(ctx context.Context) ([]string, error)
}

type MatchFactory interface {
	CreateMatch(ctx context.Context, req MatchRequest) (uuid.UUID, error)
}

// Notifier pushes match-ready signals to connected players.
type Notifier interface {
	NotifyMatchFound(ctx context.Context, userID uuid.UUID, matchID uuid.UUID) error
	NotifySoloFallback(ctx context.Context, userID uuid.UUID) error
	NotifyCrossMatch(ctx context.Context, userID uuid.UUID, targetTier string) error
}

// RematchLoader provides match details for rematch requests.
type RematchLoader interface {
	LoadForRematch(ctx context.Context, matchID uuid.UUID) (tier, topic, mode string, players []QueueEntry, err error)
}
