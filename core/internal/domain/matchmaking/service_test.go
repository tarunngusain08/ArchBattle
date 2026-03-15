package matchmaking

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

// -- Mock implementations --

type mockQueueStore struct {
	entries  []QueueEntry
	queues   []string
	dequeued map[uuid.UUID]struct{}
}

func newMockQueueStore(entries []QueueEntry) *mockQueueStore {
	queues := map[string]struct{}{}
	for _, e := range entries {
		queues["mm:"+string(e.Tier)+":"+string(e.Topic)] = struct{}{}
	}
	keys := make([]string, 0, len(queues))
	for k := range queues {
		keys = append(keys, k)
	}
	return &mockQueueStore{entries: entries, queues: keys, dequeued: map[uuid.UUID]struct{}{}}
}

func (m *mockQueueStore) Enqueue(_ context.Context, entry QueueEntry) error {
	m.entries = append(m.entries, entry)
	return nil
}
func (m *mockQueueStore) Dequeue(_ context.Context, _, _ string, userID uuid.UUID) error {
	m.dequeued[userID] = struct{}{}
	return nil
}
func (m *mockQueueStore) FindCandidates(_ context.Context, tier, topic string, minELO, maxELO int) ([]QueueEntry, error) {
	var result []QueueEntry
	for _, e := range m.entries {
		if _, ok := m.dequeued[e.UserID]; ok {
			continue
		}
		if string(e.Tier) != tier || string(e.Topic) != topic {
			continue
		}
		if e.ELO < minELO || e.ELO > maxELO {
			continue
		}
		result = append(result, e)
	}
	return result, nil
}
func (m *mockQueueStore) GetEntryTime(_ context.Context, _ uuid.UUID) (time.Time, error) {
	return time.Now().Add(-10 * time.Second), nil
}
func (m *mockQueueStore) GetAllActiveQueues(_ context.Context) ([]string, error) {
	return m.queues, nil
}

type mockMatchFactory struct {
	created []uuid.UUID
}

func (m *mockMatchFactory) CreateMatch(_ context.Context, req MatchRequest) (uuid.UUID, error) {
	if len(req.Players) < 2 {
		return uuid.Nil, errors.New("need at least 2 players")
	}
	id := uuid.New()
	m.created = append(m.created, id)
	return id, nil
}

type mockNotifier struct {
	notified []uuid.UUID
}

func (m *mockNotifier) NotifyMatchFound(_ context.Context, userID, _ uuid.UUID) error {
	m.notified = append(m.notified, userID)
	return nil
}
func (m *mockNotifier) NotifySoloFallback(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockNotifier) NotifyCrossMatch(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

// -- Tests --

func makeEntry(elo int, tier, topic string) QueueEntry {
	return QueueEntry{
		UserID:   uuid.New(),
		Username: "user",
		Tier:     shared.Tier(tier),
		Topic:    shared.Topic(topic),
		ELO:      elo,
		JoinedAt: time.Now().Add(-5 * time.Second),
	}
}

func TestRunScanOnce_FormsPair(t *testing.T) {
	entries := []QueueEntry{
		makeEntry(1000, "junior", "caching"),
		makeEntry(1050, "junior", "caching"),
	}
	queue := newMockQueueStore(entries)
	factory := &mockMatchFactory{}
	notifier := &mockNotifier{}
	svc := NewService(queue, factory, notifier)

	if err := svc.RunScanOnce(context.Background()); err != nil {
		t.Fatalf("RunScanOnce failed: %v", err)
	}
	if len(factory.created) != 1 {
		t.Errorf("expected 1 match created, got %d", len(factory.created))
	}
	if len(notifier.notified) != 2 {
		t.Errorf("expected 2 players notified, got %d", len(notifier.notified))
	}
}

func TestRunScanOnce_NoPairIfOnlyOnePlayer(t *testing.T) {
	entries := []QueueEntry{
		makeEntry(1000, "senior", "queues"),
	}
	queue := newMockQueueStore(entries)
	factory := &mockMatchFactory{}
	notifier := &mockNotifier{}
	svc := NewService(queue, factory, notifier)

	if err := svc.RunScanOnce(context.Background()); err != nil {
		t.Fatalf("RunScanOnce failed: %v", err)
	}
	if len(factory.created) != 0 {
		t.Errorf("expected 0 matches created, got %d", len(factory.created))
	}
}

func TestRunScanOnce_ELORangeFilters(t *testing.T) {
	// Two players with ELO difference >150 should NOT be paired (within 10s wait)
	entries := []QueueEntry{
		makeEntry(1000, "junior", "storage"),
		makeEntry(1200, "junior", "storage"),
	}
	queue := newMockQueueStore(entries)
	factory := &mockMatchFactory{}
	notifier := &mockNotifier{}
	svc := NewService(queue, factory, notifier)

	if err := svc.RunScanOnce(context.Background()); err != nil {
		t.Fatalf("RunScanOnce failed: %v", err)
	}
	if len(factory.created) != 0 {
		t.Errorf("expected no match due to ELO range, got %d", len(factory.created))
	}
}
