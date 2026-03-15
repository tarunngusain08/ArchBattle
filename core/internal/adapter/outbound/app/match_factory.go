package app

import (
	"context"

	"github.com/google/uuid"

	domainmatch "github.com/radhakrishna/archbattle/core/internal/domain/match"
	domainmatchmaking "github.com/radhakrishna/archbattle/core/internal/domain/matchmaking"
)

// LoopRegistrar registers the expected player count for a match so the WS gateway
// can launch the game loop once all players connect.
type LoopRegistrar interface {
	SetExpectedPlayers(matchID uuid.UUID, count int)
}

type MatchFactory struct {
	service       *domainmatch.Service
	loopRegistrar LoopRegistrar
}

func NewMatchFactory(service *domainmatch.Service, loopRegistrar LoopRegistrar) *MatchFactory {
	return &MatchFactory{service: service, loopRegistrar: loopRegistrar}
}

func (f *MatchFactory) SetLoopRegistrar(r LoopRegistrar) {
	f.loopRegistrar = r
}

func (f *MatchFactory) CreateMatch(ctx context.Context, req domainmatchmaking.MatchRequest) (uuid.UUID, error) {
	players := make([]domainmatch.PlayerProfile, 0, len(req.Players))
	for _, player := range req.Players {
		players = append(players, domainmatch.PlayerProfile{UserID: player.UserID, Username: player.Username, CurrentELO: player.ELO, MatchesPlayed: 0})
	}
	created, err := f.service.CreateMatch(ctx, domainmatch.CreateMatchRequest{Mode: req.Mode, Topic: req.Topic, Tier: req.Tier, Players: players})
	if err != nil {
		return uuid.Nil, err
	}
	if f.loopRegistrar != nil {
		f.loopRegistrar.SetExpectedPlayers(created.ID, len(req.Players))
	}
	return created.ID, nil
}
