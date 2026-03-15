package tutor

import "github.com/radhakrishna/archbattle/ai-service/internal/domain/shared"

type Request struct {
	UserID         string           `json:"userId"`
	MatchID        string           `json:"matchId"`
	QuestionID     string           `json:"questionId"`
	QuestionPrompt string           `json:"questionPrompt"`
	OfficialReason string           `json:"officialReason"`
	PlayerAnswer   []int            `json:"playerAnswer"`
	History        []shared.Message `json:"history"`
}

type Response struct {
	Text       string `json:"text"`
	TokenCount int    `json:"tokenCount"`
}
