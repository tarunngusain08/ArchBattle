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

const draftSystemPrompt = `Return valid JSON only.`

const draftExamplePrompt = `Generate a scenario-based system design MCQ for topic=%s, tier=%s.

You MUST respond with ONLY valid JSON matching this exact structure:
{
  "scenario": "A video streaming platform...",
  "prompt": "Which architecture change should be prioritized?",
  "options": ["Option A", "Option B", "Option C", "Option D"],
  "correctAnswers": [1],
  "rationale": "Because..."
}

Example:
{
  "scenario": "Your ecommerce platform experiences a flash sale where traffic increases from 3K to 120K req/sec within 30 seconds. The primary Postgres database CPU reaches 95%%.",
  "prompt": "What architecture change should be prioritized first to reduce database load?",
  "options": [
    "Add read replicas to the Postgres database",
    "Introduce Redis caching for product catalog queries",
    "Increase Postgres instance size",
    "Add additional API servers"
  ],
  "correctAnswers": [1],
  "rationale": "The workload is read-heavy, so introducing Redis caching drastically reduces DB queries."
}`

func (s *Service) Generate(ctx context.Context, req Request) (*Draft, error) {
	prompt := fmt.Sprintf(draftExamplePrompt, req.Topic, req.Tier)
	if req.Seed != "" {
		prompt += "\n\nSeed context: " + req.Seed
	}
	text, _, err := s.llm.Complete(ctx, shared.CompletionRequest{
		Model:        "gpt-4o-mini",
		SystemPrompt: draftSystemPrompt,
		Messages:     []shared.Message{{Role: "user", Content: prompt}},
		Temperature:  0.2,
		MaxTokens:    700,
	})
	if err != nil {
		return nil, err
	}
	draft := &Draft{}
	if err := json.Unmarshal([]byte(text), draft); err != nil {
		return nil, fmt.Errorf("parse draft json: %w", err)
	}
	return draft, nil
}
