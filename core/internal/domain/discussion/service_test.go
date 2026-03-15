package discussion

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockRepo struct {
	created  []CreateRequest
	entries  []*Entry
	upvoted  []uuid.UUID
	createErr error
	listErr  error
	upvoteErr error
}

func (m *mockRepo) Create(ctx context.Context, req CreateRequest) (*Entry, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.created = append(m.created, req)
	return &Entry{
		ID:            uuid.New(),
		ChallengeDate: req.ChallengeDate,
		UserID:        req.UserID,
		QuestionNumber: req.QuestionNumber,
		ReasoningText:  req.ReasoningText,
		AlternativeText: req.AlternativeText,
		SurpriseText:   req.SurpriseText,
		Upvotes:       0,
		CreatedAt:     time.Now().UTC(),
	}, nil
}

func (m *mockRepo) ListByDate(ctx context.Context, date time.Time) ([]*Entry, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.entries, nil
}

func (m *mockRepo) Upvote(ctx context.Context, entryID, voterID uuid.UUID) error {
	if m.upvoteErr != nil {
		return m.upvoteErr
	}
	m.upvoted = append(m.upvoted, entryID)
	return nil
}

func TestService_Add_ValidatesQuestionNumber(t *testing.T) {
	svc := NewService(&mockRepo{})
	ctx := context.Background()
	date := time.Now().UTC().Truncate(24 * time.Hour)

	_, err := svc.Add(ctx, CreateRequest{
		UserID:         uuid.New(),
		ChallengeDate:   date,
		QuestionNumber:  0,
		ReasoningText:   "reasoning",
	})
	if err != ErrInvalidQuestionNumber {
		t.Errorf("expected ErrInvalidQuestionNumber for question 0, got %v", err)
	}

	_, err = svc.Add(ctx, CreateRequest{
		UserID:         uuid.New(),
		ChallengeDate:   date,
		QuestionNumber:  4,
		ReasoningText:   "reasoning",
	})
	if err != ErrInvalidQuestionNumber {
		t.Errorf("expected ErrInvalidQuestionNumber for question 4, got %v", err)
	}
}

func TestService_Add_ValidatesTextLength(t *testing.T) {
	svc := NewService(&mockRepo{})
	ctx := context.Background()
	date := time.Now().UTC().Truncate(24 * time.Hour)

	longReasoning := string(make([]byte, MaxReasoningLen+1))
	_, err := svc.Add(ctx, CreateRequest{
		UserID:         uuid.New(),
		ChallengeDate:   date,
		QuestionNumber:  1,
		ReasoningText:   longReasoning,
	})
	if err != ErrTextTooLong {
		t.Errorf("expected ErrTextTooLong for long reasoning, got %v", err)
	}
}

func TestService_Add_Success(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)
	ctx := context.Background()
	date := time.Now().UTC().Truncate(24 * time.Hour)
	userID := uuid.New()

	entry, err := svc.Add(ctx, CreateRequest{
		UserID:         userID,
		ChallengeDate:   date,
		QuestionNumber:  1,
		ReasoningText:   "my reasoning",
		AlternativeText: "alternatives",
		SurpriseText:    "surprises",
	})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if len(repo.created) != 1 {
		t.Errorf("expected 1 create call, got %d", len(repo.created))
	}
	if repo.created[0].ReasoningText != "my reasoning" {
		t.Errorf("expected reasoning 'my reasoning', got %q", repo.created[0].ReasoningText)
	}
}

func TestService_List(t *testing.T) {
	entries := []*Entry{
		{ID: uuid.New(), Upvotes: 5},
		{ID: uuid.New(), Upvotes: 3},
	}
	repo := &mockRepo{entries: entries}
	svc := NewService(repo)
	ctx := context.Background()
	date := time.Now().UTC().Truncate(24 * time.Hour)

	got, err := svc.List(ctx, date)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 entries, got %d", len(got))
	}
}

func TestService_Upvote(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)
	ctx := context.Background()
	entryID := uuid.New()
	voterID := uuid.New()

	err := svc.Upvote(ctx, entryID, voterID)
	if err != nil {
		t.Fatalf("Upvote failed: %v", err)
	}
	if len(repo.upvoted) != 1 || repo.upvoted[0] != entryID {
		t.Errorf("expected upvote for entry %s, got %v", entryID, repo.upvoted)
	}
}
