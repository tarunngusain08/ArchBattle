package variation

import (
	"context"
	"strings"

	"github.com/radhakrishna/archbattle/ai-service/internal/domain/shared"
)

type Service struct {
	llm shared.LLMClient
}

func NewService(llm shared.LLMClient) *Service {
	return &Service{llm: llm}
}

func (s *Service) Generate(ctx context.Context, req Request) (*Response, error) {
	if req.Count <= 0 {
		req.Count = 3
	}
	text, _, err := s.llm.Complete(ctx, shared.CompletionRequest{Model: "gpt-4o-mini", SystemPrompt: "Produce one variation per line.", Messages: []shared.Message{{Role: "user", Content: "Create structural variations for: " + req.BasePrompt}}, Temperature: 0.4, MaxTokens: 600})
	if err != nil {
		return nil, err
	}
	lines := strings.Split(text, "\n")
	variations := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
		if line != "" {
			variations = append(variations, line)
		}
	}
	if len(variations) == 0 {
		variations = []string{req.BasePrompt}
	}
	return &Response{Variations: variations}, nil
}
