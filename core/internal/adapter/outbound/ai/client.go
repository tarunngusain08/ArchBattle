package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	stdhttp "net/http"
	"strings"
	"time"

	domainmatch "github.com/radhakrishna/archbattle/core/internal/domain/match"
	domainquestion "github.com/radhakrishna/archbattle/core/internal/domain/question"
	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

// draftQuestionPayload is the wire format sent to the AI service.
type draftQuestionPayload struct {
	Topic string `json:"topic"`
	Tier  string `json:"tier"`
	Mode  string `json:"mode"`
	Seed  string `json:"seed"`
}

type draftQuestionsPayload struct {
	Topic string `json:"topic"`
	Tier  string `json:"tier"`
	Mode  string `json:"mode"`
	Count int    `json:"count"`
}

type Client struct {
	baseURL        string
	internalSecret string
	http           *stdhttp.Client
}

func NewClient(baseURL, internalSecret string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), internalSecret: internalSecret, http: &stdhttp.Client{Timeout: 45 * time.Second}}
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

func (c *Client) DraftQuestions(ctx context.Context, topic, tier, mode string, count int) ([]map[string]any, error) {
	var response []map[string]any
	payload := draftQuestionsPayload{Topic: topic, Tier: tier, Mode: mode, Count: count}
	if err := c.postJSON(ctx, "/ai/draft-questions", payload, &response); err != nil {
		return nil, err
	}
	return response, nil
}

type aiDraftResponse struct {
	Scenario       string   `json:"scenario"`
	Prompt         string   `json:"prompt"`
	Options        []string `json:"options"`
	CorrectAnswers []int    `json:"correctAnswers"`
	Rationale      string   `json:"rationale"`
}

// GenerateQuestion implements domainquestion.AIQuestionGenerator.
func (c *Client) GenerateQuestion(ctx context.Context, topic, tier, mode string) (*domainquestion.Question, error) {
	resp, err := c.DraftQuestion(ctx, domainquestion.DraftRequest{Topic: topic, Tier: tier, Mode: mode, Seed: ""})
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("marshal draft response: %w", err)
	}
	var d aiDraftResponse
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("parse draft response: %w", err)
	}
	if err := validateAIDraft(&d); err != nil {
		return nil, err
	}
	tierParsed, err := shared.ParseTier(tier)
	if err != nil {
		tierParsed = shared.TierJunior
	}
	topicParsed := shared.NormalizeTopic(topic)
	modeParsed := shared.Mode(mode)
	if modeParsed != shared.ModeFFF {
		modeParsed = shared.ModeFFF
	}
	prompt := strings.TrimSpace(d.Scenario)
	if d.Prompt != "" {
		if prompt != "" {
			prompt += "\n\n"
		}
		prompt += strings.TrimSpace(d.Prompt)
	}
	return &domainquestion.Question{
		Prompt:         prompt,
		Options:        d.Options,
		CorrectAnswers: d.CorrectAnswers,
		Rationale:      d.Rationale,
		Topic:          topicParsed,
		DifficultyTier: tierParsed,
		Mode:           modeParsed,
	}, nil
}

func (c *Client) GenerateQuestions(ctx context.Context, topic, tier, mode string, count int) ([]*domainquestion.Question, error) {
	responses, err := c.DraftQuestions(ctx, topic, tier, mode, count)
	if err != nil {
		return nil, err
	}
	tierParsed, err := shared.ParseTier(tier)
	if err != nil {
		tierParsed = shared.TierJunior
	}
	topicParsed := shared.NormalizeTopic(topic)
	modeParsed := shared.Mode(mode)
	if modeParsed != shared.ModeFFF {
		modeParsed = shared.ModeFFF
	}
	questions := make([]*domainquestion.Question, 0, len(responses))
	for _, resp := range responses {
		data, err := json.Marshal(resp)
		if err != nil {
			continue
		}
		var d aiDraftResponse
		if err := json.Unmarshal(data, &d); err != nil {
			continue
		}
		if err := validateAIDraft(&d); err != nil {
			continue
		}
		prompt := strings.TrimSpace(d.Scenario)
		if d.Prompt != "" {
			if prompt != "" {
				prompt += "\n\n"
			}
			prompt += strings.TrimSpace(d.Prompt)
		}
		questions = append(questions, &domainquestion.Question{
			Prompt:         prompt,
			Options:        d.Options,
			CorrectAnswers: d.CorrectAnswers,
			Rationale:      d.Rationale,
			Topic:          topicParsed,
			DifficultyTier: tierParsed,
			Mode:           modeParsed,
		})
	}
	return questions, nil
}

func validateAIDraft(d *aiDraftResponse) error {
	if len(strings.Fields(d.Scenario)) < 50 {
		return fmt.Errorf("scenario too short (min 50 words)")
	}
	if len(d.Prompt) < 10 {
		return fmt.Errorf("prompt too short (min 10 chars)")
	}
	if len(d.Options) != 4 {
		return fmt.Errorf("options must have exactly 4 elements, got %d", len(d.Options))
	}
	for i, o := range d.Options {
		if strings.TrimSpace(o) == "" {
			return fmt.Errorf("option %d is empty", i)
		}
	}
	if len(d.CorrectAnswers) < 1 {
		return fmt.Errorf("correctAnswers must have at least 1 element")
	}
	for _, a := range d.CorrectAnswers {
		if a < 0 || a > 3 {
			return fmt.Errorf("correctAnswers index %d out of bounds [0,3]", a)
		}
	}
	if strings.TrimSpace(d.Rationale) == "" {
		return fmt.Errorf("rationale is empty")
	}
	return nil
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
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("ai service returned %s: %s", res.Status, string(body))
	}
	if response == nil {
		return nil
	}
	if err := json.NewDecoder(res.Body).Decode(response); err != nil {
		return fmt.Errorf("decode ai response: %w", err)
	}
	return nil
}
