package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	domainshared "github.com/radhakrishna/archbattle/ai-service/internal/domain/shared"
)

type SessionLogger struct {
	client *goredis.Client
}

func NewSessionLogger(client *goredis.Client) *SessionLogger {
	return &SessionLogger{client: client}
}

func (l *SessionLogger) Log(ctx context.Context, record domainshared.SessionRecord) error {
	payload, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal session log: %w", err)
	}
	key := fmt.Sprintf("ai:sessions:%s:%s", record.UserID, time.Now().UTC().Format("2006-01-02"))
	if err := l.client.RPush(ctx, key, payload).Err(); err != nil {
		return fmt.Errorf("push session log: %w", err)
	}
	_ = l.client.Expire(ctx, key, 7*24*time.Hour).Err()
	return nil
}
