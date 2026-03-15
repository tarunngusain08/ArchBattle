package redis

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	domainmatch "github.com/radhakrishna/archbattle/core/internal/domain/match"
)

type MatchStreamReader struct {
	events      domainmatch.EventPublisher
	broadcaster domainmatch.Broadcaster
	block       time.Duration
	started     sync.Map
}

func NewMatchStreamReader(events domainmatch.EventPublisher, broadcaster domainmatch.Broadcaster, block time.Duration) *MatchStreamReader {
	if block <= 0 {
		block = 2 * time.Second
	}
	return &MatchStreamReader{events: events, broadcaster: broadcaster, block: block}
}

func (r *MatchStreamReader) Start(ctx context.Context, matchID uuid.UUID) {
	if _, loaded := r.started.LoadOrStore(matchID.String(), struct{}{}); loaded {
		return
	}
	go func() {
		defer r.started.Delete(matchID.String())
		_ = r.events.Stream(ctx, matchID, "0", r.block, func(event *domainmatch.MatchEvent) error {
			return r.broadcaster.BroadcastToMatch(ctx, matchID, event)
		})
	}()
}
