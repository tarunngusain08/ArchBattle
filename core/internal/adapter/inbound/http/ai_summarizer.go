package http

import (
	"context"

	outboundai "github.com/radhakrishna/archbattle/core/internal/adapter/outbound/ai"
)

// AISummarizerAdapter adapts the AI client to the DiscussionSummarizer interface.
type AISummarizerAdapter struct {
	client *outboundai.Client
}

func NewAISummarizerAdapter(client *outboundai.Client) *AISummarizerAdapter {
	return &AISummarizerAdapter{client: client}
}

func (a *AISummarizerAdapter) SummarizeDiscussion(ctx context.Context, date string, entries []DiscussionEntryInput) (string, error) {
	converted := make([]outboundai.DiscussionEntrySummary, len(entries))
	for i, e := range entries {
		converted[i] = outboundai.DiscussionEntrySummary{
			QuestionNumber:   e.QuestionNumber,
			Username:         e.Username,
			ReasoningText:    e.ReasoningText,
			AlternativeText:  e.AlternativeText,
			SurpriseText:     e.SurpriseText,
		}
	}
	return a.client.SummarizeDiscussion(ctx, date, converted)
}
