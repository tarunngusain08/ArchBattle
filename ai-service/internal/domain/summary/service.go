package summary

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/radhakrishna/archbattle/ai-service/internal/domain/shared"
)

type Service struct {
	llm shared.LLMClient
}

func NewService(llm shared.LLMClient) *Service {
	return &Service{llm: llm}
}

func (s *Service) Learning(ctx context.Context, req LearningRequest) (*LearningResponse, error) {
	text, _, err := s.llm.Complete(ctx, shared.CompletionRequest{Model: "openai/gpt-4o-mini", SystemPrompt: "Return compact JSON with strength, weakness, recommendation, eloNarrative.", Messages: []shared.Message{{Role: "user", Content: fmt.Sprintf("Summarize the match %s on topic %s tier %s with standings %v", req.MatchID, req.Topic, req.Tier, req.Standings)}}, Temperature: 0.2, MaxTokens: 300})
	if err != nil {
		return nil, err
	}
	response := &LearningResponse{}
	if err := json.Unmarshal([]byte(text), response); err != nil {
		return &LearningResponse{Strength: "Fast pattern recognition", Weakness: "Needs clearer trade-off articulation", Recommendation: "Review storage, consistency, and scaling trade-offs", ELONarrative: text}, nil
	}
	return response, nil
}

func (s *Service) Discussion(ctx context.Context, req DiscussionRequest) (*DiscussionResponse, error) {
	text, _, err := s.llm.Complete(ctx, shared.CompletionRequest{Model: "openai/gpt-4o-mini", SystemPrompt: "Summarize the discussion in 3-5 sentences.", Messages: append([]shared.Message{{Role: "user", Content: req.Question}}, req.Messages...), Temperature: 0.2, MaxTokens: 250})
	if err != nil {
		return nil, err
	}
	return &DiscussionResponse{Summary: text}, nil
}
