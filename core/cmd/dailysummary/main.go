// dailysummary is a cron job that generates AI summaries for daily challenge
// discussion entries. Run at 00:05 UTC (e.g. via system cron) to process
// yesterday's entries and store the summary in daily_challenges.ai_summary.
//
// Usage:
//
//	DATABASE_URL=... AI_SERVICE_URL=... AI_INTERNAL_SECRET=... go run ./core/cmd/dailysummary
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	outboundai "github.com/radhakrishna/archbattle/core/internal/adapter/outbound/ai"
	outboundpostgres "github.com/radhakrishna/archbattle/core/internal/adapter/outbound/postgres"
	domaindaily "github.com/radhakrishna/archbattle/core/internal/domain/daily"
	domaindiscussion "github.com/radhakrishna/archbattle/core/internal/domain/discussion"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	databaseURL := env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/archbattle?sslmode=disable")
	aiServiceURL := env("AI_SERVICE_URL", "")
	aiSecret := env("AI_INTERNAL_SECRET", "")

	if aiServiceURL == "" {
		logger.Warn("AI_SERVICE_URL not set, skipping summary generation")
		return
	}

	pool, err := outboundpostgres.NewPool(ctx, databaseURL)
	if err != nil {
		logger.Error("connect postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	dateStr := yesterday.Format("2006-01-02")

	discussionRepo := outboundpostgres.NewDiscussionRepo(pool)
	dailyRepo := outboundpostgres.NewDailyRepo(pool)
	discussionService := domaindiscussion.NewService(discussionRepo)
	dailyService := domaindaily.NewService(dailyRepo, nil, nil, 48)

	entries, err := discussionService.List(ctx, yesterday)
	if err != nil {
		logger.Error("list discussion entries", "error", err, "date", dateStr)
		os.Exit(1)
	}
	if len(entries) == 0 {
		logger.Info("no discussion entries for date", "date", dateStr)
		return
	}

	aiClient := outboundai.NewClient(aiServiceURL, aiSecret)
	inputs := make([]outboundai.DiscussionEntrySummary, len(entries))
	for i, e := range entries {
		inputs[i] = outboundai.DiscussionEntrySummary{
			QuestionNumber:   e.QuestionNumber,
			Username:         e.Username,
			ReasoningText:    e.ReasoningText,
			AlternativeText:  e.AlternativeText,
			SurpriseText:     e.SurpriseText,
		}
	}

	summary, err := aiClient.SummarizeDiscussion(ctx, dateStr, inputs)
	if err != nil {
		logger.Error("ai summarize failed", "error", err)
		os.Exit(1)
	}

	if err := dailyService.UpdateAISummary(ctx, yesterday, summary); err != nil {
		logger.Error("update ai summary failed", "error", err)
		os.Exit(1)
	}

	logger.Info("daily summary generated", "date", dateStr, "entries_count", len(entries))
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
