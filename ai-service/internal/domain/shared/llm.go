package shared

import (
	"context"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CompletionRequest struct {
	Model        string    `json:"model"`
	SystemPrompt string    `json:"systemPrompt"`
	Messages     []Message `json:"messages"`
	Temperature  float64   `json:"temperature"`
	MaxTokens    int       `json:"maxTokens"`
}

type LLMClient interface {
	Complete(ctx context.Context, req CompletionRequest) (string, int, error)
}

type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, ttl time.Duration) (bool, int64, error)
}

type SessionRecord struct {
	Kind       string    `json:"kind"`
	UserID     string    `json:"userId"`
	MatchID    string    `json:"matchId,omitempty"`
	QuestionID string    `json:"questionId,omitempty"`
	Messages   []Message `json:"messages"`
	TokenCount int       `json:"tokenCount"`
	CreatedAt  time.Time `json:"createdAt"`
}

type SessionLogger interface {
	Log(ctx context.Context, record SessionRecord) error
}
