package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	stdhttp "net/http"
	"os"
	"strings"
	"time"

	domainshared "github.com/radhakrishna/archbattle/ai-service/internal/domain/shared"
)

type Client struct {
	apiKey string
	http   *stdhttp.Client
}

type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	System      string             `json:"system,omitempty"`
	Temperature float64            `json:"temperature,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string          `json:"role"`
	Content []anthropicText `json:"content"`
}

type anthropicText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicResponse struct {
	Content []anthropicText `json:"content"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func NewClient(apiKey string) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	return &Client{apiKey: apiKey, http: &stdhttp.Client{Timeout: 45 * time.Second}}
}

func (c *Client) Complete(ctx context.Context, req domainshared.CompletionRequest) (string, int, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return "", 0, fmt.Errorf("ANTHROPIC_API_KEY is not configured")
	}

	messages := make([]anthropicMessage, 0, len(req.Messages))
	for _, message := range req.Messages {
		messages = append(messages, anthropicMessage{Role: message.Role, Content: []anthropicText{{Type: "text", Text: message.Content}}})
	}
	payload := anthropicRequest{Model: req.Model, MaxTokens: req.MaxTokens, System: req.SystemPrompt, Temperature: req.Temperature, Messages: messages}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", 0, fmt.Errorf("marshal anthropic request: %w", err)
	}

	httpReq, err := stdhttp.NewRequestWithContext(ctx, stdhttp.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", 0, fmt.Errorf("create anthropic request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	res, err := c.http.Do(httpReq)
	if err != nil {
		return "", 0, fmt.Errorf("execute anthropic request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return "", 0, fmt.Errorf("anthropic returned %s", res.Status)
	}

	response := anthropicResponse{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return "", 0, fmt.Errorf("decode anthropic response: %w", err)
	}
	parts := make([]string, 0, len(response.Content))
	for _, block := range response.Content {
		if block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n"), response.Usage.InputTokens + response.Usage.OutputTokens, nil
}

