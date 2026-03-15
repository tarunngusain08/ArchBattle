package cron

import (
	"context"
	"log/slog"
	"time"

	domaindaily "github.com/radhakrishna/archbattle/core/internal/domain/daily"
)

type Publisher interface {
	Publish(ctx context.Context, date time.Time) (*domaindaily.DailyChallenge, error)
}

type FreezeCrediter interface {
	CreditWeeklyFreeze(ctx context.Context) error
}

type Scheduler struct {
	publisher     Publisher
	freezeCrediter FreezeCrediter
	logger       *slog.Logger
}

func NewScheduler(publisher Publisher, freezeCrediter FreezeCrediter, logger *slog.Logger) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Scheduler{publisher: publisher, freezeCrediter: freezeCrediter, logger: logger}
}

func (s *Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	lastPublishedDate := time.Time{}
	lastFreezeCredit := time.Time{}
	for {
		now := time.Now().UTC()
		if (lastPublishedDate.IsZero() || !sameDay(lastPublishedDate, now)) && now.Hour() == 0 {
			if _, err := s.publisher.Publish(ctx, now); err != nil {
				s.logger.Error("publish daily challenge", "error", err)
			} else {
				lastPublishedDate = now
			}
		}
		if s.freezeCrediter != nil && now.Weekday() == time.Monday && now.Hour() == 9 &&
			(lastFreezeCredit.IsZero() || !sameDay(lastFreezeCredit, now)) {
			if err := s.freezeCrediter.CreditWeeklyFreeze(ctx); err != nil {
				s.logger.Error("credit weekly freeze", "error", err)
			} else {
				lastFreezeCredit = now
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func sameDay(left, right time.Time) bool {
	left = left.UTC()
	right = right.UTC()
	return left.Year() == right.Year() && left.YearDay() == right.YearDay()
}
