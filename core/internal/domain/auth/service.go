package auth

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

var (
	usernameRegex        = regexp.MustCompile(`^[a-zA-Z0-9_]{3,30}$`)
	emailRegex           = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
)

type Service struct {
	users    UserRepository
	sessions SessionStore
	hasher   PasswordHasher
	tokens   TokenIssuer
	ttl      time.Duration
}

func NewService(users UserRepository, sessions SessionStore, hasher PasswordHasher, tokens TokenIssuer, ttl time.Duration) *Service {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	return &Service{users: users, sessions: sessions, hasher: hasher, tokens: tokens, ttl: ttl}
}

func (s *Service) Register(ctx context.Context, username, email, password string) (*AuthResult, error) {
	username = strings.TrimSpace(username)
	email = strings.ToLower(strings.TrimSpace(email))
	if username == "" || email == "" || len(password) < 8 {
		return nil, fmt.Errorf("username, email and an 8+ char password are required")
	}
	if !usernameRegex.MatchString(username) {
		return nil, fmt.Errorf("username must be 3-30 chars, alphanumeric and underscore only")
	}
	if !emailRegex.MatchString(email) {
		return nil, fmt.Errorf("invalid email format")
	}

	if existing, err := s.users.FindByEmail(ctx, email); err == nil && existing != nil {
		return nil, fmt.Errorf("email already exists")
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &User{ID: uuid.New(), Username: username, Email: email, PasswordHash: hash, Role: RoleUser, Tier: shared.TierJunior, JuniorELO: 1000, SeniorELO: 1000, StaffELO: 1000, CreatedAt: time.Now().UTC()}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return s.newAuthResult(ctx, user)
}

func (s *Service) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.users.FindByEmail(ctx, email)
	if err != nil || user == nil {
		return nil, ErrInvalidCredentials
	}
	if err := s.hasher.Compare(user.PasswordHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}
	return s.newAuthResult(ctx, user)
}

func (s *Service) Authenticate(ctx context.Context, rawToken string) (*Session, error) {
	token := strings.TrimSpace(rawToken)
	if token == "" {
		return nil, ErrInvalidToken
	}

	parsed, err := s.tokens.Parse(token)
	if err != nil {
		return nil, ErrInvalidToken
	}
	session, err := s.sessions.Get(ctx, token)
	if err != nil || session == nil || session.UserID != parsed.UserID {
		return nil, ErrInvalidToken
	}
	session.Token = token
	return session, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	return s.sessions.Delete(ctx, strings.TrimSpace(token))
}

func (s *Service) GetUser(ctx context.Context, userID uuid.UUID) (*User, error) {
	return s.users.FindByID(ctx, userID)
}

func (s *Service) newAuthResult(ctx context.Context, user *User) (*AuthResult, error) {
	role := user.Role
	if role == "" {
		role = RoleUser
	}
	session := &Session{UserID: user.ID, Username: user.Username, Role: role, Tier: user.Tier, ELO: user.CurrentELO(user.Tier), ExpiresAt: time.Now().UTC().Add(s.ttl)}
	token, err := s.tokens.Issue(session, s.ttl)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}
	session.Token = token
	if err := s.sessions.Set(ctx, token, session, s.ttl); err != nil {
		return nil, fmt.Errorf("store session: %w", err)
	}
	return &AuthResult{User: user, Session: session}, nil
}
