package room

import (
	"context"

	"github.com/google/uuid"
)

type RoomStore interface {
	Set(ctx context.Context, code string, matchID, ownerID uuid.UUID) error
	Get(ctx context.Context, code string) (matchID, ownerID uuid.UUID, err error)
}

type RoomStatus struct {
	MatchID     uuid.UUID `json:"matchId"`
	PlayerCount int       `json:"playerCount"`
	Status      string    `json:"status"`
}
