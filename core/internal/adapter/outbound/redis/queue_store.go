package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	domainmatchmaking "github.com/radhakrishna/archbattle/core/internal/domain/matchmaking"
)

type QueueStore struct {
	client *goredis.Client
}

func NewQueueStore(client *goredis.Client) *QueueStore {
	return &QueueStore{client: client}
}

func (s *QueueStore) Enqueue(ctx context.Context, entry domainmatchmaking.QueueEntry) error {
	key := queueKey(string(entry.Tier), string(entry.Topic))
	payload, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encode queue entry: %w", err)
	}
	if err := s.client.ZAdd(ctx, key, goredis.Z{Score: float64(entry.ELO), Member: entry.UserID.String()}).Err(); err != nil {
		return fmt.Errorf("zadd queue entry: %w", err)
	}
	entryKey := queueEntryKey(entry.UserID)
	if err := s.client.Set(ctx, entryKey, payload, 120*time.Second).Err(); err != nil {
		return fmt.Errorf("set queue entry ttl: %w", err)
	}
	return nil
}

func (s *QueueStore) Dequeue(ctx context.Context, tier string, topic string, userID uuid.UUID) error {
	if err := s.client.ZRem(ctx, queueKey(tier, topic), userID.String()).Err(); err != nil {
		return fmt.Errorf("remove queue member: %w", err)
	}
	_ = s.client.Del(ctx, queueEntryKey(userID)).Err()
	return nil
}

func (s *QueueStore) FindCandidates(ctx context.Context, tier string, topic string, minELO, maxELO int) ([]domainmatchmaking.QueueEntry, error) {
	members, err := s.client.ZRangeByScoreWithScores(ctx, queueKey(tier, topic), &goredis.ZRangeBy{Min: fmt.Sprintf("%d", minELO), Max: fmt.Sprintf("%d", maxELO)}).Result()
	if err != nil {
		return nil, fmt.Errorf("zrange queue candidates: %w", err)
	}
	entries := make([]domainmatchmaking.QueueEntry, 0, len(members))
	for _, member := range members {
		userID, err := uuid.Parse(member.Member.(string))
		if err != nil {
			continue
		}
		raw, err := s.client.Get(ctx, queueEntryKey(userID)).Result()
		if err != nil {
			continue
		}
		var entry domainmatchmaking.QueueEntry
		if err := json.Unmarshal([]byte(raw), &entry); err != nil {
			return nil, fmt.Errorf("decode queue entry: %w", err)
		}
		entry.ELO = int(member.Score)
		entries = append(entries, entry)
	}
	return entries, nil
}

func (s *QueueStore) GetEntry(ctx context.Context, userID uuid.UUID) (*domainmatchmaking.QueueEntry, error) {
	raw, err := s.client.Get(ctx, queueEntryKey(userID)).Result()
	if err != nil {
		return nil, fmt.Errorf("read queue entry: %w", err)
	}
	var entry domainmatchmaking.QueueEntry
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		return nil, fmt.Errorf("decode queue entry: %w", err)
	}
	return &entry, nil
}

func (s *QueueStore) GetEntryTime(ctx context.Context, userID uuid.UUID) (time.Time, error) {
	entry, err := s.GetEntry(ctx, userID)
	if err != nil {
		return time.Time{}, err
	}
	return entry.JoinedAt, nil
}

func (s *QueueStore) GetAllActiveQueues(ctx context.Context) ([]string, error) {
	cursor := uint64(0)
	keys := []string{}
	for {
		batch, next, err := s.client.Scan(ctx, cursor, "queue:*:*", 50).Result()
		if err != nil {
			return nil, fmt.Errorf("scan queues: %w", err)
		}
		for _, key := range batch {
			if strings.HasPrefix(key, "queue:entry:") {
				continue
			}
			parts := strings.Split(key, ":")
			if len(parts) == 3 {
				keys = append(keys, key)
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

func queueKey(tier string, topic string) string {
	return fmt.Sprintf("queue:%s:%s", tier, topic)
}

func queueEntryKey(userID uuid.UUID) string {
	return fmt.Sprintf("queue:entry:%s", userID)
}
