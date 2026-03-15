package leaderboard

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// RecordELO stores the player's new absolute ELO on both global and weekly leaderboards.
func (s *Service) RecordELO(ctx context.Context, tier shared.Tier, userID uuid.UUID, absoluteELO int, at time.Time) error {
	week := shared.CurrentWeekKey(at)
	if err := s.store.Set(ctx, tier, ScopeGlobal, "", userID, absoluteELO); err != nil {
		return err
	}
	return s.store.Set(ctx, tier, ScopeWeekly, week, userID, absoluteELO)
}

func (s *Service) List(ctx context.Context, tier shared.Tier, scope Scope, limit int, now time.Time) ([]Entry, error) {
	week := ""
	if scope == ScopeWeekly {
		week = shared.CurrentWeekKey(now)
	}
	return s.store.Top(ctx, tier, scope, week, limit)
}

func (s *Service) Rank(ctx context.Context, tier shared.Tier, scope Scope, userID uuid.UUID, now time.Time) (int64, error) {
	week := ""
	if scope == ScopeWeekly {
		week = shared.CurrentWeekKey(now)
	}
	return s.store.Rank(ctx, tier, scope, week, userID)
}
