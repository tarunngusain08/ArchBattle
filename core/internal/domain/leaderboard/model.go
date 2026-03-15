package leaderboard

import (
	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type Scope string

const (
	ScopeGlobal Scope = "global"
	ScopeWeekly Scope = "weekly"
)

type Entry struct {
	UserID   uuid.UUID   `json:"userId"`
	Tier     shared.Tier `json:"tier"`
	Scope    Scope       `json:"scope"`
	Week     string      `json:"week,omitempty"`
	Score    float64     `json:"score"`
	Rank     int64       `json:"rank"`
	Username string      `json:"username,omitempty"`
}
