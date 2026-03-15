package auth

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	UpdateELO(ctx context.Context, id uuid.UUID, tier shared.Tier, newELO int) error
	UpdateStreak(ctx context.Context, id uuid.UUID, streak int, lastDate time.Time) error
	IncrementMatchesPlayed(ctx context.Context, id uuid.UUID) error
}

type SessionStore interface {
	Set(ctx context.Context, token string, session *Session, ttl time.Duration) error
	Get(ctx context.Context, token string) (*Session, error)
	Delete(ctx context.Context, token string) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type TokenIssuer interface {
	Issue(session *Session, ttl time.Duration) (string, error)
	Parse(token string) (*Session, error)
}
