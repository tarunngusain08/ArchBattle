package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

// -- Mock implementations --

type mockUserRepo struct {
	users map[string]*User
}

func newMockUserRepo() *mockUserRepo { return &mockUserRepo{users: map[string]*User{}} }

func (m *mockUserRepo) Create(_ context.Context, user *User) error {
	m.users[user.Email] = user
	return nil
}
func (m *mockUserRepo) FindByEmail(_ context.Context, email string) (*User, error) {
	if u, ok := m.users[email]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}
func (m *mockUserRepo) FindByID(_ context.Context, id uuid.UUID) (*User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, errors.New("not found")
}
func (m *mockUserRepo) UpdateELO(_ context.Context, _ uuid.UUID, _ shared.Tier, _ int) error {
	return nil
}
func (m *mockUserRepo) UpdateStreak(_ context.Context, _ uuid.UUID, _ int, _ time.Time) error {
	return nil
}
func (m *mockUserRepo) IncrementMatchesPlayed(_ context.Context, _ uuid.UUID) error { return nil }

type mockSessionStore struct {
	sessions map[string]*Session
}

func newMockSessionStore() *mockSessionStore { return &mockSessionStore{sessions: map[string]*Session{}} }

func (m *mockSessionStore) Set(_ context.Context, token string, session *Session, _ time.Duration) error {
	m.sessions[token] = session
	return nil
}
func (m *mockSessionStore) Get(_ context.Context, token string) (*Session, error) {
	if s, ok := m.sessions[token]; ok {
		return s, nil
	}
	return nil, nil
}
func (m *mockSessionStore) Delete(_ context.Context, token string) error {
	delete(m.sessions, token)
	return nil
}

type mockHasher struct{}

func (mockHasher) Hash(p string) (string, error)        { return "hashed:" + p, nil }
func (mockHasher) Compare(hash, p string) error {
	if hash == "hashed:"+p {
		return nil
	}
	return errors.New("mismatch")
}

type mockTokenIssuer struct{ counter int }

func (m *mockTokenIssuer) Issue(session *Session, _ time.Duration) (string, error) {
	m.counter++
	token := "token-" + session.UserID.String()
	return token, nil
}
func (m *mockTokenIssuer) Parse(token string) (*Session, error) {
	// Expect "token-<uuid>"
	if len(token) < 7 {
		return nil, errors.New("invalid")
	}
	id, err := uuid.Parse(token[6:])
	if err != nil {
		return nil, err
	}
	return &Session{UserID: id}, nil
}

// -- Tests --

func buildService() (*Service, *mockUserRepo, *mockSessionStore) {
	repo := newMockUserRepo()
	store := newMockSessionStore()
	svc := NewService(repo, store, mockHasher{}, &mockTokenIssuer{}, time.Hour)
	return svc, repo, store
}

func TestRegister(t *testing.T) {
	svc, _, _ := buildService()
	result, err := svc.Register(context.Background(), "alice", "alice@example.com", "password123")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if result.User.Username != "alice" {
		t.Errorf("expected username alice, got %s", result.User.Username)
	}
	if result.Session.Token == "" {
		t.Error("expected non-empty token")
	}
	if result.User.Role != RoleUser {
		t.Errorf("expected role %q, got %q", RoleUser, result.User.Role)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	svc, _, _ := buildService()
	_, err := svc.Register(context.Background(), "alice", "alice@example.com", "password123")
	if err != nil {
		t.Fatalf("first Register failed: %v", err)
	}
	_, err = svc.Register(context.Background(), "alice2", "alice@example.com", "password123")
	if err == nil {
		t.Error("expected error for duplicate email")
	}
}

func TestLogin(t *testing.T) {
	svc, _, _ := buildService()
	_, err := svc.Register(context.Background(), "bob", "bob@example.com", "securepass")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	result, err := svc.Login(context.Background(), "bob@example.com", "securepass")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if result.User.Username != "bob" {
		t.Errorf("expected username bob, got %s", result.User.Username)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	svc, _, _ := buildService()
	_, _ = svc.Register(context.Background(), "carol", "carol@example.com", "rightpassword")
	_, err := svc.Login(context.Background(), "carol@example.com", "wrongpassword")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthenticate(t *testing.T) {
	svc, _, _ := buildService()
	result, _ := svc.Register(context.Background(), "dan", "dan@example.com", "passphrase1")
	token := result.Session.Token

	session, err := svc.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}
	if session.UserID != result.User.ID {
		t.Errorf("user ID mismatch")
	}
}
