package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	stdhttp "net/http"
	"strings"
	"time"

	domainmatch "github.com/radhakrishna/archbattle/core/internal/domain/match"
	domainquestion "github.com/radhakrishna/archbattle/core/internal/domain/question"
)

// draftQuestionPayload is the wire format sent to the AI service.
type draftQuestionPayload struct {
	Topic string `json:"topic"`
	Tier  string `json:"tier"`
	Mode  string `json:"mode"`
	Seed  string `json:"seed"`
}

type Client struct {
	baseURL        string
	internalSecret string
	http           *stdhttp.Client
}

func NewClient(baseURL, internalSecret string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), internalSecret: internalSecret, http: &stdhttp.Client{Timeout: 20 * time.Second}}
}

func (c *Client) RequestLearningSummary(ctx context.Context, req domainmatch.LearningSummaryRequest) (*domainmatch.LearningSummary, error) {
	var summary domainmatch.LearningSummary
	if err := c.postJSON(ctx, "/ai/learning-summary", req, &summary); err != nil {
		return nil, err
	}
	return &summary, nil
}

func (c *Client) DraftQuestion(ctx context.Context, req domainquestion.DraftRequest) (map[string]any, error) {
	response := map[string]any{}
	payload := draftQuestionPayload{Topic: req.Topic, Tier: req.Tier, Mode: req.Mode, Seed: req.Seed}
	if err := c.postJSON(ctx, "/ai/draft-question", payload, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) Tutor(ctx context.Context, body map[string]any) (map[string]any, error) {
	response := map[string]any{}
	if err := c.postJSON(ctx, "/ai/tutor", body, &response); err != nil {
		return nil, err
	}
	return response, nil
}

// DiscussionSummaryInput is the payload for summarizing discussion entries.
type DiscussionSummaryInput struct {
	Date    string                   `json:"date"`
	Entries []DiscussionEntrySummary `json:"entries"`
}

type DiscussionEntrySummary struct {
	QuestionNumber   int    `json:"questionNumber"`
	Username         string `json:"username"`
	ReasoningText    string `json:"reasoningText"`
	AlternativeText  string `json:"alternativeText"`
	SurpriseText     string `json:"surpriseText"`
}

type discussionSummaryResponse struct {
	Summary string `json:"summary"`
}

// SummarizeDiscussion calls the AI service to generate a summary of discussion entries.
func (c *Client) SummarizeDiscussion(ctx context.Context, date string, entries []DiscussionEntrySummary) (string, error) {
	var resp discussionSummaryResponse
	if err := c.postJSON(ctx, "/ai/discussion-summary", DiscussionSummaryInput{Date: date, Entries: entries}, &resp); err != nil {
		return "", err
	}
	return resp.Summary, nil
}

func (c *Client) postJSON(ctx context.Context, path string, request any, response any) error {
	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal ai request: %w", err)
	}
	req, err := stdhttp.NewRequestWithContext(ctx, stdhttp.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create ai request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.internalSecret != "" {
		req.Header.Set("Authorization", "Bearer "+c.internalSecret)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("execute ai request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("ai service returned %s", res.Status)
	}
	if response == nil {
		return nil
	}
	if err := json.NewDecoder(res.Body).Decode(response); err != nil {
		return fmt.Errorf("decode ai response: %w", err)
	}
	return nil
}
