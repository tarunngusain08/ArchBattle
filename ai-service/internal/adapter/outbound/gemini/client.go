package gemini

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

const baseURL = "https://generativelanguage.googleapis.com/v1beta/models"

// Client implements shared.LLMClient using Google's Gemini REST API.
type Client struct {
	apiKey string
	http   *stdhttp.Client
}

type geminiRequest struct {
	Contents          []geminiContent    `json:"contents"`
	SystemInstruction *geminiContent     `json:"systemInstruction,omitempty"`
	GenerationConfig  geminiGenConfig    `json:"generationConfig"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenConfig struct {
	Temperature  float64 `json:"temperature"`
	MaxOutputTokens int  `json:"maxOutputTokens"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

func NewClient(apiKey string) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	return &Client{apiKey: apiKey, http: &stdhttp.Client{Timeout: 45 * time.Second}}
}

func (c *Client) Complete(ctx context.Context, req domainshared.CompletionRequest) (string, int, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return "", 0, fmt.Errorf("GEMINI_API_KEY is not configured")
	}

	contents := make([]geminiContent, 0, len(req.Messages))
	for _, msg := range req.Messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: msg.Content}},
		})
	}

	payload := geminiRequest{
		Contents:         contents,
		GenerationConfig: geminiGenConfig{Temperature: req.Temperature, MaxOutputTokens: req.MaxTokens},
	}
	if req.SystemPrompt != "" {
		payload.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: req.SystemPrompt}},
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", 0, fmt.Errorf("marshal gemini request: %w", err)
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", baseURL, req.Model, c.apiKey)
	httpReq, err := stdhttp.NewRequestWithContext(ctx, stdhttp.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", 0, fmt.Errorf("create gemini request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(httpReq)
	if err != nil {
		return "", 0, fmt.Errorf("execute gemini request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		respBody, _ := io.ReadAll(res.Body)
		return "", 0, fmt.Errorf("gemini returned %s: %s", res.Status, string(respBody))
	}

	var response geminiResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return "", 0, fmt.Errorf("decode gemini response: %w", err)
	}

	var parts []string
	for _, candidate := range response.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				parts = append(parts, part.Text)
			}
		}
	}
	return strings.Join(parts, "\n"), response.UsageMetadata.TotalTokenCount, nil
}
