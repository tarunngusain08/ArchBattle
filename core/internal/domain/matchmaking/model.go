package matchmaking

import (
	"time"

	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type QueueEntry struct {
	UserID   uuid.UUID    `json:"userId"`
	Username string       `json:"username"`
	Tier     shared.Tier  `json:"tier"`
	Topic    shared.Topic `json:"topic"`
	Mode     shared.Mode  `json:"mode"`
	ELO      int          `json:"elo"`
	JoinedAt time.Time    `json:"joinedAt"`
}

type MatchRequest struct {
	Players []QueueEntry `json:"players"`
	Tier    shared.Tier  `json:"tier"`
	Topic   shared.Topic `json:"topic"`
	Mode    shared.Mode  `json:"mode"`
	Solo    bool         `json:"solo"`
}
