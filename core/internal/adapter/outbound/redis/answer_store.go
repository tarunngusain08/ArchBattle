package redis

import (
	"context"
	"encoding/json"
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

func (s *AnswerStore) StoreLatestChoice(ctx context.Context, matchID, qID, userID uuid.UUID, choices []int) error {
	data, err := json.Marshal(choices)
	if err != nil {
		return fmt.Errorf("marshal choices: %w", err)
	}
	if err := s.client.HSet(ctx, latestChoiceKey(matchID, qID), userID.String(), data).Err(); err != nil {
		return fmt.Errorf("store latest choice: %w", err)
	}
	return nil
}

func (s *AnswerStore) GetAllLatestChoices(ctx context.Context, matchID, qID uuid.UUID) (map[uuid.UUID][]int, error) {
	raw, err := s.client.HGetAll(ctx, latestChoiceKey(matchID, qID)).Result()
	if err != nil {
		return nil, fmt.Errorf("read latest choices: %w", err)
	}
	result := make(map[uuid.UUID][]int, len(raw))
	for k, v := range raw {
		uid, err := uuid.Parse(k)
		if err != nil {
			continue
		}
		var choices []int
		if err := json.Unmarshal([]byte(v), &choices); err != nil {
			continue
		}
		result[uid] = choices
	}
	return result, nil
}

func (s *AnswerStore) RecordFirstAnswerTime(ctx context.Context, matchID, qID, userID uuid.UUID, score float64) (bool, error) {
	added, err := s.client.ZAddArgs(ctx, answerKey(matchID, qID), goredis.ZAddArgs{
		NX:      true,
		Members: []goredis.Z{{Score: score, Member: userID.String()}},
	}).Result()
	if err != nil {
		return false, fmt.Errorf("zadd first answer time: %w", err)
	}
	return added == 1, nil
}

func (s *AnswerStore) ClearQuestion(ctx context.Context, matchID, qID uuid.UUID) error {
	if err := s.client.Del(ctx, answerKey(matchID, qID), answerSeqKey(matchID, qID), latestChoiceKey(matchID, qID)).Err(); err != nil {
		return fmt.Errorf("clear answer keys: %w", err)
	}
	return nil
}

func (s *AnswerStore) SetQuestionTTL(ctx context.Context, matchID, qID uuid.UUID, ttl time.Duration) error {
	pipe := s.client.Pipeline()
	pipe.Expire(ctx, answerKey(matchID, qID), ttl)
	pipe.Expire(ctx, answerSeqKey(matchID, qID), ttl)
	pipe.Expire(ctx, latestChoiceKey(matchID, qID), ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func answerKey(matchID, qID uuid.UUID) string {
	return fmt.Sprintf("match:%s:q:%s:answers", matchID, qID)
}

func answerSeqKey(matchID, qID uuid.UUID) string {
	return fmt.Sprintf("match:%s:q:%s:seq", matchID, qID)
}

func latestChoiceKey(matchID, qID uuid.UUID) string {
	return fmt.Sprintf("match:%s:q:%s:latest", matchID, qID)
}
