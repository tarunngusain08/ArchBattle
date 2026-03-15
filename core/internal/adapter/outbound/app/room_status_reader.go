package app

import (
	"context"

	"github.com/google/uuid"

	domainmatch "github.com/radhakrishna/archbattle/core/internal/domain/match"
	domainroom "github.com/radhakrishna/archbattle/core/internal/domain/room"
)

// Ensure RoomStatusReader implements domainroom.RoomStatusReader
var _ domainroom.RoomStatusReader = (*RoomStatusReader)(nil)

type RoomStatusReader struct {
	matchRepo domainmatch.MatchRepository
	stateStore domainmatch.MatchStateStore
}

func NewRoomStatusReader(matchRepo domainmatch.MatchRepository, stateStore domainmatch.MatchStateStore) *RoomStatusReader {
	return &RoomStatusReader{matchRepo: matchRepo, stateStore: stateStore}
}

func (r *RoomStatusReader) GetRoomStatus(ctx context.Context, matchID uuid.UUID) (playerCount int, status string, err error) {
	match, err := r.matchRepo.FindByID(ctx, matchID)
	if err != nil {
		return 0, "", err
	}
	if match == nil {
		return 0, "", nil
	}
	state, err := r.stateStore.GetMatchState(ctx, matchID)
	if err != nil {
		return 0, "", err
	}
	if state != nil {
		playerCount = len(state.PlayerIDs)
	}
	return playerCount, string(match.Status), nil
}
