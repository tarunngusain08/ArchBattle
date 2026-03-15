package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	domaindaily "github.com/radhakrishna/archbattle/core/internal/domain/daily"
)

type DailyCacheStore struct {
	client *goredis.Client
}

func NewDailyCacheStore(client *goredis.Client) *DailyCacheStore {
	return &DailyCacheStore{client: client}
}

func (s *DailyCacheStore) SetChallenge(ctx context.Context, challenge *domaindaily.DailyChallenge, ttl time.Duration) error {
	payload, err := json.Marshal(challenge)
	if err != nil {
		return fmt.Errorf("encode daily challenge: %w", err)
	}
	if err := s.client.Set(ctx, dailyKey(challenge.ChallengeDate), payload, ttl).Err(); err != nil {
		return fmt.Errorf("cache daily challenge: %w", err)
	}
	return nil
}

func (s *DailyCacheStore) GetChallenge(ctx context.Context, date time.Time) (*domaindaily.DailyChallenge, error) {
	raw, err := s.client.Get(ctx, dailyKey(date)).Result()
	if err == goredis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get cached daily challenge: %w", err)
	}
	challenge := &domaindaily.DailyChallenge{}
	if err := json.Unmarshal([]byte(raw), challenge); err != nil {
		return nil, fmt.Errorf("decode cached daily challenge: %w", err)
	}
	return challenge, nil
}

func (s *DailyCacheStore) DeleteBoard(ctx context.Context, date time.Time) error {
	if err := s.client.Del(ctx, dailyBoardKey(date)).Err(); err != nil {
		return fmt.Errorf("delete daily leaderboard cache: %w", err)
	}
	return nil
}

func dailyKey(date time.Time) string {
	return fmt.Sprintf("daily:%s", date.UTC().Format("2006-01-02"))
}
