package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	domainmatch "github.com/radhakrishna/archbattle/core/internal/domain/match"
	domainmatchmaking "github.com/radhakrishna/archbattle/core/internal/domain/matchmaking"
)

type RematchLoader struct {
	matchRepo domainmatch.MatchRepository
}

func NewRematchLoader(matchRepo domainmatch.MatchRepository) *RematchLoader {
	return &RematchLoader{matchRepo: matchRepo}
}

func (l *RematchLoader) LoadForRematch(ctx context.Context, matchID uuid.UUID) (tier, topic, mode string, players []domainmatchmaking.QueueEntry, err error) {
	match, err := l.matchRepo.FindByID(ctx, matchID)
	if err != nil || match == nil {
		return "", "", "", nil, fmt.Errorf("match not found: %w", err)
	}
	playerRows, err := l.matchRepo.GetPlayers(ctx, matchID)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("get players: %w", err)
	}
	entries := make([]domainmatchmaking.QueueEntry, 0, len(playerRows))
	for _, row := range playerRows {
		elo := row.ELOAfter
		if elo == 0 {
			elo = row.ELOBefore
		}
		entries = append(entries, domainmatchmaking.QueueEntry{
			UserID:   row.UserID,
			Username: row.Username,
			ELO:      elo,
		})
	}
	return string(match.Tier), string(match.Topic), string(match.Mode), entries, nil
}
