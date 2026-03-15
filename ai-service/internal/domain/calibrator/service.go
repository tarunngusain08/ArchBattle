package calibrator

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/radhakrishna/archbattle/ai-service/internal/domain/shared"
)

type Service struct {
	llm shared.LLMClient
}

func NewService(llm shared.LLMClient) *Service {
	return &Service{llm: llm}
}

func (s *Service) Calibrate(ctx context.Context, req Request) (*Response, error) {
	text, _, err := s.llm.Complete(ctx, shared.CompletionRequest{Model: "gemini-2.0-flash", SystemPrompt: "Return exactly: <difficulty>|<reasoning> where difficulty is 1-5.", Messages: []shared.Message{{Role: "user", Content: req.Prompt}}, Temperature: 0.1, MaxTokens: 100})
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(strings.TrimSpace(text), "|", 2)
	if len(parts) != 2 {
		return &Response{Difficulty: 3, Reasoning: text}, nil
	}
	difficulty, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return nil, fmt.Errorf("parse difficulty: %w", err)
	}
	return &Response{Difficulty: difficulty, Reasoning: strings.TrimSpace(parts[1])}, nil
}
