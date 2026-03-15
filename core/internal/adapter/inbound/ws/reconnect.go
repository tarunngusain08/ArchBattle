package ws

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	domainmatch "github.com/radhakrishna/archbattle/core/internal/domain/match"
)

func (g *Gateway) handleReconnect(ctx context.Context, c *client, raw json.RawMessage) error {
	var payload reconnectPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	matchID, err := uuid.Parse(payload.MatchID)
	if err != nil {
		return err
	}

	events, err := g.events.ReadEvents(ctx, matchID, payload.LastSeq)
	if err != nil {
		return err
	}
	for _, event := range events {
		if payload.LastSeq != "" && event.Sequence == payload.LastSeq {
			continue
		}
		if err := g.SendToPlayer(ctx, c.userID, event); err != nil {
			return err
		}
	}

	g.attachClientToMatch(c, matchID)
	if err := g.matchService.SetConnected(context.Background(), matchID, c.userID); err != nil {
		return err
	}
	return g.SendToPlayer(ctx, c.userID, &domainmatch.MatchEvent{Type: "reconnect_complete", MatchID: matchID, CreatedAt: time.Now().UTC(), Payload: map[string]any{"current_state": "active"}})
}
