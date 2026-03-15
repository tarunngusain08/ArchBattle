package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const roomTTL = 30 * time.Minute

type RoomStore struct {
	client *goredis.Client
}

func NewRoomStore(client *goredis.Client) *RoomStore {
	return &RoomStore{client: client}
}

func (s *RoomStore) Set(ctx context.Context, code string, matchID, ownerID uuid.UUID) error {
	key := fmt.Sprintf("room:%s", code)
	pipe := s.client.Pipeline()
	pipe.HSet(ctx, key, "matchId", matchID.String(), "ownerId", ownerID.String())
	pipe.Expire(ctx, key, roomTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("set room code: %w", err)
	}
	return nil
}

func (s *RoomStore) Get(ctx context.Context, code string) (uuid.UUID, uuid.UUID, error) {
	key := fmt.Sprintf("room:%s", code)
	vals, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("get room code: %w", err)
	}
	if len(vals) == 0 {
		return uuid.Nil, uuid.Nil, nil
	}
	matchID, err := uuid.Parse(vals["matchId"])
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("parse match id: %w", err)
	}
	ownerID, _ := uuid.Parse(vals["ownerId"])
	return matchID, ownerID, nil
}
