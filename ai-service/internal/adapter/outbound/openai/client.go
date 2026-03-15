package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	stdhttp "net/http"
	"os"
	"strings"
	"time"

	domainshared "github.com/radhakrishna/archbattle/ai-service/internal/domain/shared"
)

const baseURL = "https://api.openai.com/v1/chat/completions"

// Client implements shared.LLMClient using OpenAI's Chat Completions API.
type Client struct {
	apiKey string
	http   *stdhttp.Client
}

type openAIRequest struct {
	Model        string          `json:"model"`
	Messages     []openAIMessage  `json:"messages"`
	Temperature  float64         `json:"temperature"`
	MaxTokens    int             `json:"max_tokens"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// NewClient creates an OpenAI client. If apiKey is empty, uses OPENAI_API_KEY env.
func NewClient(apiKey string) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	return &Client{apiKey: apiKey, http: &stdhttp.Client{Timeout: 45 * time.Second}}
}

// Complete implements shared.LLMClient.
func (c *Client) Complete(ctx context.Context, req domainshared.CompletionRequest) (string, int, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return "", 0, fmt.Errorf("OPENAI_API_KEY is not configured")
	}

	messages := make([]openAIMessage, 0, len(req.Messages)+1)
	if req.SystemPrompt != "" {
		messages = append(messages, openAIMessage{Role: "system", Content: req.SystemPrompt})
	}
	for _, msg := range req.Messages {
		role := msg.Role
		if role == "model" {
			role = "assistant"
		}
		messages = append(messages, openAIMessage{Role: role, Content: msg.Content})
	}

	payload := openAIRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", 0, fmt.Errorf("marshal openai request: %w", err)
	}

	httpReq, err := stdhttp.NewRequestWithContext(ctx, stdhttp.MethodPost, baseURL, bytes.NewReader(body))
	if err != nil {
		return "", 0, fmt.Errorf("create openai request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	res, err := c.http.Do(httpReq)
	if err != nil {
		return "", 0, fmt.Errorf("execute openai request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		respBody, _ := io.ReadAll(res.Body)
		return "", 0, fmt.Errorf("openai returned %s: %s", res.Status, string(respBody))
	}

	var response openAIResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return "", 0, fmt.Errorf("decode openai response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", 0, fmt.Errorf("openai returned no choices")
	}
	text := response.Choices[0].Message.Content
	return strings.TrimSpace(text), response.Usage.TotalTokens, nil
}
