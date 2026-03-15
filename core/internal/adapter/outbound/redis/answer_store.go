package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type AnswerStore struct {
	client *goredis.Client
}

func NewAnswerStore(client *goredis.Client) *AnswerStore {
	return &AnswerStore{client: client}
}

func (s *AnswerStore) RecordAnswer(ctx context.Context, matchID, qID, userID uuid.UUID, score float64) (bool, error) {
	added, err := s.client.ZAddArgs(ctx, answerKey(matchID, qID), goredis.ZAddArgs{NX: true, Members: []goredis.Z{{Score: score, Member: userID.String()}}}).Result()
	if err != nil {
		return false, fmt.Errorf("zadd answer ordering: %w", err)
	}
	return added == 1, nil
}

func (s *AnswerStore) IncrementSeq(ctx context.Context, matchID, qID uuid.UUID) (int64, error) {
	value, err := s.client.Incr(ctx, answerSeqKey(matchID, qID)).Result()
	if err != nil {
		return 0, fmt.Errorf("increment answer sequence: %w", err)
	}
	return value, nil
}

func (s *AnswerStore) GetRank(ctx context.Context, matchID, qID, userID uuid.UUID) (int64, error) {
	rank, err := s.client.ZRank(ctx, answerKey(matchID, qID), userID.String()).Result()
	if err != nil {
		return 0, fmt.Errorf("read answer rank: %w", err)
	}
	return rank, nil
}

func (s *AnswerStore) GetTotalAnswered(ctx context.Context, matchID, qID uuid.UUID) (int64, error) {
	total, err := s.client.ZCard(ctx, answerKey(matchID, qID)).Result()
	if err != nil {
		return 0, fmt.Errorf("count answers: %w", err)
	}
	return total, nil
}

func (s *AnswerStore) ClearQuestion(ctx context.Context, matchID, qID uuid.UUID) error {
	if err := s.client.Del(ctx, answerKey(matchID, qID), answerSeqKey(matchID, qID)).Err(); err != nil {
		return fmt.Errorf("clear answer keys: %w", err)
	}
	return nil
}

func (s *AnswerStore) SetQuestionTTL(ctx context.Context, matchID, qID uuid.UUID, ttl time.Duration) error {
	pipe := s.client.Pipeline()
	pipe.Expire(ctx, answerKey(matchID, qID), ttl)
	pipe.Expire(ctx, answerSeqKey(matchID, qID), ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func answerKey(matchID, qID uuid.UUID) string {
	return fmt.Sprintf("match:%s:q:%s:answers", matchID, qID)
}

func answerSeqKey(matchID, qID uuid.UUID) string {
	return fmt.Sprintf("match:%s:q:%s:seq", matchID, qID)
}
