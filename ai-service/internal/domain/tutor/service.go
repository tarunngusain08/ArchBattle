package tutor

import (
	"context"
	"fmt"
	"time"

	"github.com/radhakrishna/archbattle/ai-service/internal/domain/shared"
)

type Service struct {
	llm       LLMClient
	limiter   RateLimiter
	logger    SessionLogger
	freeLimit int
}

func NewService(llm LLMClient, limiter RateLimiter, logger SessionLogger, freeLimit int) *Service {
	if freeLimit <= 0 {
		freeLimit = 3
	}
	return &Service{llm: llm, limiter: limiter, logger: logger, freeLimit: freeLimit}
}

func (s *Service) Handle(ctx context.Context, req Request) (*Response, error) {
	allowed, _, err := s.limiter.Allow(ctx, fmt.Sprintf("ratelimit:tutor:%s:%s", req.UserID, time.Now().UTC().Format("2006-01-02")), s.freeLimit, 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("apply tutor rate limit: %w", err)
	}
	if !allowed {
		return nil, fmt.Errorf("rate limit exceeded")
	}

	prompt := "You are the ArchBattle tutor. Stay grounded to the official rationale, explain trade-offs clearly, and avoid contradicting the canonical answer unless explicitly framing contextual nuance."
	// History contains prior turns; the new user message is appended at the end so
	// the conversation flows chronologically: earliest turns first, newest last.
	messages := append(req.History, shared.Message{Role: "user", Content: fmt.Sprintf("Question: %s\nOfficial rationale: %s\nPlayer answer: %v", req.QuestionPrompt, req.OfficialReason, req.PlayerAnswer)})
	text, tokens, err := s.llm.Complete(ctx, shared.CompletionRequest{Model: "claude-3-5-haiku-latest", SystemPrompt: prompt, Messages: messages, Temperature: 0.3, MaxTokens: 500})
	if err != nil {
		return nil, fmt.Errorf("generate tutor response: %w", err)
	}
	_ = s.logger.Log(ctx, shared.SessionRecord{Kind: "tutor", UserID: req.UserID, MatchID: req.MatchID, QuestionID: req.QuestionID, Messages: messages, TokenCount: tokens, CreatedAt: time.Now().UTC()})
	return &Response{Text: text, TokenCount: tokens}, nil
}
