package summary

import "github.com/radhakrishna/archbattle/ai-service/internal/domain/shared"

type LearningRequest struct {
	MatchID   string `json:"matchId"`
	Topic     string `json:"topic"`
	Tier      string `json:"tier"`
	Standings []any  `json:"standings"`
}

type LearningResponse struct {
	Strength       string `json:"strength"`
	Weakness       string `json:"weakness"`
	Recommendation string `json:"recommendation"`
	ELONarrative   string `json:"eloNarrative"`
}

type DiscussionRequest struct {
	Question string           `json:"question"`
	Messages []shared.Message `json:"messages"`
}

type DiscussionResponse struct {
	Summary string `json:"summary"`
}
