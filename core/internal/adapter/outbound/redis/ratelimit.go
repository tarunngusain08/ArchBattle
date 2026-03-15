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
		return false, 0, fmt.Errorf("increment rate limit key: %w", err)
	}
	if count == 1 {
		if err := r.client.Expire(ctx, key, ttl).Err(); err != nil {
			return false, 0, fmt.Errorf("set rate limit ttl: %w", err)
		}
	}
	return count <= int64(limit), count, nil
}

func (r *RateLimiter) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("increment key: %w", err)
	}
	if count == 1 {
		if err := r.client.Expire(ctx, key, ttl).Err(); err != nil {
			return 0, fmt.Errorf("set increment ttl: %w", err)
		}
	}
	return count, nil
}
