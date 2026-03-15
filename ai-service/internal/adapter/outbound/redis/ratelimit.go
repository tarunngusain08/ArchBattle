package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client *goredis.Client
}

func NewRateLimiter(client *goredis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

func (r *RateLimiter) Allow(ctx context.Context, key string, limit int, ttl time.Duration) (bool, int64, error) {
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return false, 0, fmt.Errorf("increment rate limit: %w", err)
	}
	if count == 1 {
		if err := r.client.Expire(ctx, key, ttl).Err(); err != nil {
			return false, 0, fmt.Errorf("expire rate limit key: %w", err)
		}
	}
	return count <= int64(limit), count, nil
}
