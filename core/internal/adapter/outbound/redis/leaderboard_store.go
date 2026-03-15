package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	domainleaderboard "github.com/radhakrishna/archbattle/core/internal/domain/leaderboard"
	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type LeaderboardStore struct {
	client *goredis.Client
}

func NewLeaderboardStore(client *goredis.Client) *LeaderboardStore {
	return &LeaderboardStore{client: client}
}

// Set stores the player's absolute ELO as the sorted set score, overwriting any prior value.
func (s *LeaderboardStore) Set(ctx context.Context, tier shared.Tier, scope domainleaderboard.Scope, week string, userID uuid.UUID, absoluteELO int) error {
	key := leaderboardKey(tier, scope, week)
	if err := s.client.ZAdd(ctx, key, goredis.Z{Score: float64(absoluteELO), Member: userID.String()}).Err(); err != nil {
		return fmt.Errorf("zadd leaderboard: %w", err)
	}
	if scope == domainleaderboard.ScopeWeekly {
		_ = s.client.Expire(ctx, key, 7*24*time.Hour).Err()
	}
	return nil
}

func (s *LeaderboardStore) Top(ctx context.Context, tier shared.Tier, scope domainleaderboard.Scope, week string, limit int) ([]domainleaderboard.Entry, error) {
	values, err := s.client.ZRevRangeWithScores(ctx, leaderboardKey(tier, scope, week), 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("zrevrange leaderboard: %w", err)
	}
	entries := make([]domainleaderboard.Entry, 0, len(values))
	for idx, value := range values {
		userID, err := uuid.Parse(value.Member.(string))
		if err != nil {
			continue
		}
		entries = append(entries, domainleaderboard.Entry{UserID: userID, Tier: tier, Scope: scope, Week: week, Score: value.Score, Rank: int64(idx + 1)})
	}
	return entries, nil
}

func (s *LeaderboardStore) Rank(ctx context.Context, tier shared.Tier, scope domainleaderboard.Scope, week string, userID uuid.UUID) (int64, error) {
	rank, err := s.client.ZRevRank(ctx, leaderboardKey(tier, scope, week), userID.String()).Result()
	if err == goredis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("zrevrank leaderboard: %w", err)
	}
	return rank + 1, nil
}

func leaderboardKey(tier shared.Tier, scope domainleaderboard.Scope, week string) string {
	if scope == domainleaderboard.ScopeWeekly {
		return fmt.Sprintf("lb:weekly:%s:%s", tier, week)
	}
	return fmt.Sprintf("lb:global:%s", tier)
}
