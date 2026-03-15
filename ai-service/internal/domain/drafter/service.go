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

The scenario MUST be at least 50 words. Make the scenario challenging and confusing with realistic constraints and red herrings.

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
  "scenario": "Your ecommerce platform experiences a flash sale where traffic increases from 3K to 120K req/sec within 30 seconds. The primary Postgres database CPU reaches 95%%. Product catalog reads dominate traffic. The inventory service is also under load. You have limited time before checkout failures spike. Multiple teams are proposing different fixes. Make the scenario challenging and confusing with realistic constraints and red herrings.",
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

const bulkDraftPrompt = `Generate exactly %d scenario-based system design MCQs for topic=%s, tier=%s.

Each scenario MUST be at least 50 words. Make scenarios challenging and confusing with realistic constraints and red herrings.

You MUST respond with ONLY a valid JSON array of objects. Each object:
{
  "scenario": "At least 50 words of context...",
  "prompt": "The question text",
  "options": ["Option A", "Option B", "Option C", "Option D"],
  "correctAnswers": [0-3],
  "rationale": "Explanation"
}

Return ONLY the array, e.g. [{"scenario":"...","prompt":"...","options":[...],"correctAnswers":[...],"rationale":"..."}, ...]`

func (s *Service) Generate(ctx context.Context, req Request) (*Draft, error) {
	prompt := fmt.Sprintf(draftExamplePrompt, req.Topic, req.Tier)
	if req.Seed != "" {
		prompt += "\n\nSeed context: " + req.Seed
	}
	text, _, err := s.llm.Complete(ctx, shared.CompletionRequest{
		Model:        "openai/gpt-4o-mini",
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

func (s *Service) GenerateBulk(ctx context.Context, req BulkRequest) ([]Draft, error) {
	if req.Count <= 0 {
		req.Count = 5
	}
	prompt := fmt.Sprintf(bulkDraftPrompt, req.Count, req.Topic, req.Tier)
	text, _, err := s.llm.Complete(ctx, shared.CompletionRequest{
		Model:        "openai/gpt-4o-mini",
		SystemPrompt: draftSystemPrompt,
		Messages:     []shared.Message{{Role: "user", Content: prompt}},
		Temperature:  0.2,
		MaxTokens:    4000,
	})
	if err != nil {
		return nil, err
	}
	var drafts []Draft
	if err := json.Unmarshal([]byte(text), &drafts); err != nil {
		return nil, fmt.Errorf("parse bulk draft json: %w", err)
	}
	return drafts, nil
}
