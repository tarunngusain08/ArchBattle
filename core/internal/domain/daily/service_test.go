package daily

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestService_GenerateShareCard(t *testing.T) {
	svc := NewService(nil, nil, nil, 48)
	date := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)

	result := Result{
		UserID:         uuid.New(),
		ChallengeDate:  date,
		Score:          267,
		Percentile:     8,
		StreakDay:      5,
		ShareCardText:  "",
		CompletedAt:    time.Now().UTC(),
	}

	card := svc.GenerateShareCard(result, 2, 3)
	if card == "" {
		t.Fatal("expected non-empty share card")
	}
	if len(card) < 20 {
		t.Errorf("share card too short: %q", card)
	}
	if !strings.Contains(card, "ArchBattle Daily") {
		t.Errorf("expected share card to contain 'ArchBattle Daily', got %q", card)
	}
	if !strings.Contains(card, "archbattle.io") {
		t.Errorf("expected share card to contain 'archbattle.io', got %q", card)
	}
	if !strings.Contains(card, "267/300") {
		t.Errorf("expected share card to contain score 267/300, got %q", card)
	}
}

func TestService_BufferDays(t *testing.T) {
	ctx := context.Background()
	fromDate := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)

	type mockRepo struct {
		bufferDays int
		err       error
	}
	repo := &struct {
		bufferDays int
		err       error
	}{bufferDays: 7, err: nil}

	// We need a repo that implements BufferDays. Check the interface.
	type bufferRepo interface {
		BufferDays(ctx context.Context, fromDate time.Time) (int, error)
	}

	// Create a minimal mock that implements DailyChallengeRepository's BufferDays
	mock := &mockDailyRepo{bufferDays: 14}
	svc := NewService(mock, nil, nil, 48)

	days, err := svc.BufferDays(ctx, fromDate)
	if err != nil {
		t.Fatalf("BufferDays failed: %v", err)
	}
	if days != 14 {
		t.Errorf("expected 14 buffer days, got %d", days)
	}
	_ = repo
}

type mockDailyRepo struct {
	bufferDays int
}

func (m *mockDailyRepo) GetByDate(ctx context.Context, date time.Time) (*DailyChallenge, error) {
	return nil, nil
}
func (m *mockDailyRepo) GetPlayerResult(ctx context.Context, userID uuid.UUID, date time.Time) (*Result, error) {
	return nil, nil
}
func (m *mockDailyRepo) GetUserStreak(ctx context.Context, userID uuid.UUID) (*Streak, error) {
	return nil, nil
}
func (m *mockDailyRepo) GetStreakFreezeAvailable(ctx context.Context, userID uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockDailyRepo) UpdateUserStreak(ctx context.Context, userID uuid.UUID, streak Streak) error {
	return nil
}
func (m *mockDailyRepo) ConsumeStreakFreeze(ctx context.Context, userID uuid.UUID) error {
	return nil
}
func (m *mockDailyRepo) SavePlayerResult(ctx context.Context, result Result) error {
	return nil
}
func (m *mockDailyRepo) BufferDays(ctx context.Context, fromDate time.Time) (int, error) {
	return m.bufferDays, nil
}
func (m *mockDailyRepo) UpdateAISummary(ctx context.Context, date time.Time, summary string) error {
	return nil
}
func (m *mockDailyRepo) CreditWeeklyFreeze(ctx context.Context) error {
	return nil
}
