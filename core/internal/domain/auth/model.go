package auth

import (
	"time"

	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type User struct {
	ID            uuid.UUID   `json:"id"`
	Username      string      `json:"username"`
	Email         string      `json:"email"`
	PasswordHash  string      `json:"-"`
	Role          string      `json:"role"`
	Tier          shared.Tier `json:"tier"`
	JuniorELO     int         `json:"juniorElo"`
	SeniorELO     int         `json:"seniorElo"`
	StaffELO      int         `json:"staffElo"`
	MatchesPlayed int         `json:"matchesPlayed"`
	CurrentStreak int         `json:"currentStreak"`
	LongestStreak int         `json:"longestStreak"`
	LastDailyDate *time.Time  `json:"lastDailyDate,omitempty"`
	CreatedAt     time.Time   `json:"createdAt"`
}

type Session struct {
	Token     string      `json:"token"`
	UserID    uuid.UUID   `json:"userId"`
	Username  string      `json:"username"`
	Role      string      `json:"role"`
	Tier      shared.Tier `json:"tier"`
	ELO       int         `json:"elo"`
	ExpiresAt time.Time   `json:"expiresAt"`
}

type AuthResult struct {
	User    *User    `json:"user"`
	Session *Session `json:"session"`
}

func (u User) CurrentELO(tier shared.Tier) int {
	switch tier {
	case shared.TierSenior:
		return u.SeniorELO
	case shared.TierStaff:
		return u.StaffELO
	default:
		return u.JuniorELO
	}
}

func (u *User) SetELO(tier shared.Tier, elo int) {
	switch tier {
	case shared.TierSenior:
		u.SeniorELO = elo
	case shared.TierStaff:
		u.StaffELO = elo
	default:
		u.JuniorELO = elo
	}
}
