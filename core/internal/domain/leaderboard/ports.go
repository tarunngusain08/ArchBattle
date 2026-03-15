package leaderboard

import (
	"context"

	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type Store interface {
	// Set records the player's absolute ELO score on the leaderboard sorted set.
	Set(ctx context.Context, tier shared.Tier, scope Scope, week string, userID uuid.UUID, absoluteELO int) error
	Top(ctx context.Context, tier shared.Tier, scope Scope, week string, limit int) ([]Entry, error)
	Rank(ctx context.Context, tier shared.Tier, scope Scope, week string, userID uuid.UUID) (int64, error)
}
