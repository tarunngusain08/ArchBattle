package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	domainmatch "github.com/radhakrishna/archbattle/core/internal/domain/match"
)

type EventPublisher struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewEventPublisher(client *goredis.Client, ttl time.Duration) *EventPublisher {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &EventPublisher{client: client, ttl: ttl}
}

func (p *EventPublisher) Publish(ctx context.Context, matchID uuid.UUID, event *domainmatch.MatchEvent) error {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("encode event payload: %w", err)
	}
	key := eventStreamKey(matchID)
	if _, err := p.client.XAdd(ctx, &goredis.XAddArgs{Stream: key, MaxLen: 1000, Approx: true, Values: map[string]any{"type": event.Type, "payload": payload, "created_at": event.CreatedAt.Format(time.RFC3339Nano)}}).Result(); err != nil {
		return fmt.Errorf("xadd event: %w", err)
	}
	if err := p.client.Expire(ctx, key, p.ttl).Err(); err != nil {
		return fmt.Errorf("expire event stream: %w", err)
	}
	return nil
}

func (p *EventPublisher) ReadEvents(ctx context.Context, matchID uuid.UUID, fromSeq string) ([]*domainmatch.MatchEvent, error) {
	start := fromSeq
	if start == "" {
		start = "-"
	}
	messages, err := p.client.XRange(ctx, eventStreamKey(matchID), start, "+").Result()
	if err != nil {
		return nil, fmt.Errorf("xrange events: %w", err)
	}
	return decodeMessages(matchID, messages)
}

func (p *EventPublisher) Stream(ctx context.Context, matchID uuid.UUID, lastID string, block time.Duration, handler func(*domainmatch.MatchEvent) error) error {
	if lastID == "" {
		lastID = "$"
	}
	if block <= 0 {
		block = 2 * time.Second
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		result, err := p.client.XRead(ctx, &goredis.XReadArgs{Streams: []string{eventStreamKey(matchID), lastID}, Count: 10, Block: block}).Result()
		if err == goredis.Nil {
			continue
		}
		if err != nil {
			return fmt.Errorf("xread events: %w", err)
		}
		for _, stream := range result {
			events, err := decodeMessages(matchID, stream.Messages)
			if err != nil {
				return err
			}
			for _, event := range events {
				lastID = event.Sequence
				if err := handler(event); err != nil {
					slog.Warn("stream handler error", "match", matchID, "error", err)
					continue
				}
			}
		}
	}
}

func decodeMessages(matchID uuid.UUID, messages []goredis.XMessage) ([]*domainmatch.MatchEvent, error) {
	events := make([]*domainmatch.MatchEvent, 0, len(messages))
	for _, message := range messages {
		event := &domainmatch.MatchEvent{Sequence: message.ID, MatchID: matchID}
		if value, ok := message.Values["type"].(string); ok {
			event.Type = value
		}
		if value, ok := message.Values["created_at"].(string); ok {
			event.CreatedAt, _ = time.Parse(time.RFC3339Nano, value)
		}
		event.Payload = map[string]any{}
		if raw, ok := message.Values["payload"].(string); ok && raw != "" {
			if err := json.Unmarshal([]byte(raw), &event.Payload); err != nil {
				return nil, fmt.Errorf("decode event payload: %w", err)
			}
		}
		events = append(events, event)
	}
	return events, nil
}

func eventStreamKey(matchID uuid.UUID) string {
	return fmt.Sprintf("match:%s:events", matchID)
}
