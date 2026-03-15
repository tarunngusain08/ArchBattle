package redis

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	domainleaderboard "github.com/radhakrishna/archbattle/core/internal/domain/leaderboard"
)

type DailyLeaderboardStore struct {
	client *goredis.Client
}

func NewDailyLeaderboardStore(client *goredis.Client) *DailyLeaderboardStore {
	return &DailyLeaderboardStore{client: client}
}

func (s *DailyLeaderboardStore) AddScore(ctx context.Context, date time.Time, userID uuid.UUID, score int) error {
	key := dailyBoardKey(date)
	if err := s.client.ZAdd(ctx, key, goredis.Z{Score: float64(score), Member: userID.String()}).Err(); err != nil {
		return fmt.Errorf("zadd daily board: %w", err)
	}
	_ = s.client.Expire(ctx, key, 24*time.Hour).Err()
	return nil
}

func (s *DailyLeaderboardStore) Percentile(ctx context.Context, date time.Time, userID uuid.UUID) (float64, error) {
	key := dailyBoardKey(date)
	rank, err := s.client.ZRevRank(ctx, key, userID.String()).Result()
	if err == goredis.Nil {
		return 100, nil
	}
	if err != nil {
		return 0, fmt.Errorf("zrevrank daily board: %w", err)
	}
	total, err := s.client.ZCard(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("zcard daily board: %w", err)
	}
	if total == 0 {
		return 100, nil
	}
	return math.Ceil((float64(rank+1) / float64(total)) * 100), nil
}

func (s *DailyLeaderboardStore) Top(ctx context.Context, date time.Time, limit int) ([]domainleaderboard.Entry, error) {
	values, err := s.client.ZRevRangeWithScores(ctx, dailyBoardKey(date), 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("zrevrange daily board: %w", err)
	}
	entries := make([]domainleaderboard.Entry, 0, len(values))
	for idx, value := range values {
		userID, err := uuid.Parse(value.Member.(string))
		if err != nil {
			continue
		}
		entries = append(entries, domainleaderboard.Entry{UserID: userID, Score: value.Score, Rank: int64(idx + 1)})
	}
	return entries, nil
}

func dailyBoardKey(date time.Time) string {
	return fmt.Sprintf("daily:board:%s", date.UTC().Format("2006-01-02"))
}
