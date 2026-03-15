package tutor

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/radhakrishna/archbattle/ai-service/internal/domain/shared"
)

// -- Mock implementations --

type mockLLM struct {
	// capturedMessages records the message list from the last Complete call.
	capturedMessages []shared.Message
}

func (m *mockLLM) Complete(_ context.Context, req shared.CompletionRequest) (string, int, error) {
	m.capturedMessages = req.Messages
	return "tutor says: " + req.Messages[len(req.Messages)-1].Content, 100, nil
}

type mockRateLimiter struct{ allow bool }

func (m *mockRateLimiter) Allow(_ context.Context, _ string, _ int, _ time.Duration) (bool, int64, error) {
	return m.allow, 0, nil
}

type mockSessionLogger struct{}

func (m *mockSessionLogger) Log(_ context.Context, _ shared.SessionRecord) error { return nil }

// -- Tests --

func TestHandle_MessageOrder(t *testing.T) {
	llm := &mockLLM{}
	svc := NewService(llm, &mockRateLimiter{allow: true}, &mockSessionLogger{}, 10)

	history := []shared.Message{
		{Role: "user", Content: "prior user message"},
		{Role: "assistant", Content: "prior assistant response"},
	}

	req := Request{
		UserID:         "user-1",
		QuestionPrompt: "When would you use event sourcing?",
		OfficialReason: "Event sourcing provides an audit log.",
		PlayerAnswer:   []int{1},
		History:        history,
	}

	_, err := svc.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	msgs := llm.capturedMessages
	// History messages should come first, new user message last.
	if len(msgs) < 3 {
		t.Fatalf("expected at least 3 messages, got %d", len(msgs))
	}
	if msgs[0].Content != "prior user message" {
		t.Errorf("first message should be prior history, got %q", msgs[0].Content)
	}
	if msgs[1].Content != "prior assistant response" {
		t.Errorf("second message should be prior assistant response, got %q", msgs[1].Content)
	}
	last := msgs[len(msgs)-1]
	if last.Role != "user" {
		t.Errorf("last message should be user, got %q", last.Role)
	}
	if !strings.Contains(last.Content, "When would you use event sourcing?") {
		t.Errorf("last message should contain the question prompt, got %q", last.Content)
	}
}

func TestHandle_RateLimitExceeded(t *testing.T) {
	llm := &mockLLM{}
	svc := NewService(llm, &mockRateLimiter{allow: false}, &mockSessionLogger{}, 3)

	_, err := svc.Handle(context.Background(), Request{UserID: "user-2"})
	if err == nil {
		t.Error("expected rate limit error, got nil")
	}
	if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("error should mention rate limit, got %q", err.Error())
	}
}
