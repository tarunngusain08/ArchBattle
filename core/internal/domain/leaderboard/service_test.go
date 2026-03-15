package leaderboard

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type mockStore struct {
	setCalls  []setCall
	topResult []Entry
	topErr    error
	rankResult int64
	rankErr   error
}

type setCall struct {
	tier   shared.Tier
	scope  Scope
	week   string
	userID uuid.UUID
	elo    int
}

func (m *mockStore) Set(ctx context.Context, tier shared.Tier, scope Scope, week string, userID uuid.UUID, absoluteELO int) error {
	m.setCalls = append(m.setCalls, setCall{tier, scope, week, userID, absoluteELO})
	return nil
}

func (m *mockStore) Top(ctx context.Context, tier shared.Tier, scope Scope, week string, limit int) ([]Entry, error) {
	if m.topErr != nil {
		return nil, m.topErr
	}
	return m.topResult, nil
}

func (m *mockStore) Rank(ctx context.Context, tier shared.Tier, scope Scope, week string, userID uuid.UUID) (int64, error) {
	if m.rankErr != nil {
		return 0, m.rankErr
	}
	return m.rankResult, nil
}

func TestService_RecordELO(t *testing.T) {
	store := &mockStore{}
	svc := NewService(store)
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now().UTC()

	err := svc.RecordELO(ctx, shared.TierJunior, userID, 1200, now)
	if err != nil {
		t.Fatalf("RecordELO failed: %v", err)
	}
	if len(store.setCalls) != 2 {
		t.Errorf("expected 2 set calls (global + weekly), got %d", len(store.setCalls))
	}
	globalFound, weeklyFound := false, false
	for _, c := range store.setCalls {
		if c.scope == ScopeGlobal {
			globalFound = true
			if c.elo != 1200 {
				t.Errorf("global ELO expected 1200, got %d", c.elo)
			}
		}
		if c.scope == ScopeWeekly {
			weeklyFound = true
		}
	}
	if !globalFound || !weeklyFound {
		t.Errorf("expected both global and weekly sets, got global=%v weekly=%v", globalFound, weeklyFound)
	}
}

func TestService_List(t *testing.T) {
	entries := []Entry{
		{UserID: uuid.New(), Username: "alice", Score: 1500},
		{UserID: uuid.New(), Username: "bob", Score: 1400},
	}
	store := &mockStore{topResult: entries}
	svc := NewService(store)
	ctx := context.Background()
	now := time.Now().UTC()

	got, err := svc.List(ctx, shared.TierSenior, ScopeGlobal, 10, now)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 entries, got %d", len(got))
	}
	if got[0].Username != "alice" {
		t.Errorf("expected first entry alice, got %s", got[0].Username)
	}
}

func TestService_List_PropagatesError(t *testing.T) {
	store := &mockStore{topErr: errors.New("store error")}
	svc := NewService(store)
	ctx := context.Background()
	now := time.Now().UTC()

	_, err := svc.List(ctx, shared.TierJunior, ScopeGlobal, 10, now)
	if err == nil {
		t.Fatal("expected error from List")
	}
}

func TestService_Rank(t *testing.T) {
	store := &mockStore{rankResult: 5}
	svc := NewService(store)
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now().UTC()

	rank, err := svc.Rank(ctx, shared.TierStaff, ScopeWeekly, userID, now)
	if err != nil {
		t.Fatalf("Rank failed: %v", err)
	}
	if rank != 5 {
		t.Errorf("expected rank 5, got %d", rank)
	}
}
