package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	domainauth "github.com/radhakrishna/archbattle/core/internal/domain/auth"
	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type SessionStore struct {
	client *goredis.Client
}

func NewSessionStore(client *goredis.Client) *SessionStore {
	return &SessionStore{client: client}
}

func (s *SessionStore) Set(ctx context.Context, token string, session *domainauth.Session, ttl time.Duration) error {
	key := sessionKey(token)
	values := map[string]any{
		"uid":        session.UserID.String(),
		"username":   session.Username,
		"role":       session.Role,
		"tier":       string(session.Tier),
		"elo":        session.ELO,
		"expires_at": session.ExpiresAt.Format(time.RFC3339Nano),
	}
	if err := s.client.HSet(ctx, key, values).Err(); err != nil {
		return fmt.Errorf("write session hash: %w", err)
	}
	if err := s.client.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("set session ttl: %w", err)
	}
	return nil
}

func (s *SessionStore) Get(ctx context.Context, token string) (*domainauth.Session, error) {
	values, err := s.client.HGetAll(ctx, sessionKey(token)).Result()
	if err != nil {
		return nil, fmt.Errorf("get session hash: %w", err)
	}
	if len(values) == 0 {
		return nil, nil
	}
	userID, err := uuid.Parse(values["uid"])
	if err != nil {
		return nil, fmt.Errorf("parse session uid: %w", err)
	}
	elo, err := strconv.Atoi(values["elo"])
	if err != nil {
		return nil, fmt.Errorf("parse session elo: %w", err)
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, values["expires_at"])
	if err != nil {
		return nil, fmt.Errorf("parse session expiresAt: %w", err)
	}
	return &domainauth.Session{
		UserID:    userID,
		Username:  values["username"],
		Role:      values["role"],
		Tier:      shared.Tier(values["tier"]),
		ELO:       elo,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *SessionStore) Delete(ctx context.Context, token string) error {
	if err := s.client.Del(ctx, sessionKey(token)).Err(); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func sessionKey(token string) string {
	return fmt.Sprintf("session:%s", token)
}
