package drafter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/radhakrishna/archbattle/ai-service/internal/domain/shared"
)

type Service struct {
	llm LLMClient
}

func NewService(llm LLMClient) *Service {
	return &Service{llm: llm}
}

func (s *Service) Generate(ctx context.Context, req Request) (*Draft, error) {
	prompt := fmt.Sprintf("Generate one ArchBattle multiple-choice question for topic=%s tier=%s mode=%s. Seed context: %s. Respond with compact JSON containing prompt, options, correctAnswers, rationale.", req.Topic, req.Tier, req.Mode, req.Seed)
	text, _, err := s.llm.Complete(ctx, shared.CompletionRequest{Model: "claude-3-5-sonnet-latest", SystemPrompt: "Return valid JSON only.", Messages: []shared.Message{{Role: "user", Content: prompt}}, Temperature: 0.2, MaxTokens: 700})
	if err != nil {
		return nil, err
	}
	draft := &Draft{}
	if err := json.Unmarshal([]byte(text), draft); err != nil {
		return &Draft{Prompt: fmt.Sprintf("Draft for %s", req.Topic), Options: []string{"Option A", "Option B", "Option C", "Option D"}, CorrectAnswers: []int{0}, Rationale: text}, nil
	}
	return draft, nil
}
