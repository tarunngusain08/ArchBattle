package daily

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/match"
)

type Service struct {
	repo        DailyChallengeRepository
	cache       DailyCacheStore
	leaderboard DailyLeaderboardStore
	graceHours  int
}

func NewService(repo DailyChallengeRepository, cache DailyCacheStore, leaderboard DailyLeaderboardStore, graceHours int) *Service {
	if graceHours <= 0 {
		graceHours = 48
	}
	return &Service{repo: repo, cache: cache, leaderboard: leaderboard, graceHours: graceHours}
}

func (s *Service) Publish(ctx context.Context, date time.Time) (*DailyChallenge, error) {
	challenge, err := s.repo.GetByDate(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("load challenge: %w", err)
	}
	if challenge == nil {
		return nil, fmt.Errorf("daily challenge not found")
	}
	if err := s.cache.SetChallenge(ctx, challenge, 24*time.Hour); err != nil {
		return nil, fmt.Errorf("cache challenge: %w", err)
	}
	_ = s.cache.DeleteBoard(ctx, date.AddDate(0, 0, -1))
	return challenge, nil
}

func (s *Service) GetChallenge(ctx context.Context, date time.Time) (*DailyChallenge, error) {
	cached, err := s.cache.GetChallenge(ctx, date)
	if err == nil && cached != nil {
		return cached, nil
	}
	challenge, err := s.repo.GetByDate(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("load challenge: %w", err)
	}
	if challenge == nil {
		return nil, fmt.Errorf("daily challenge not found")
	}
	_ = s.cache.SetChallenge(ctx, challenge, 24*time.Hour)
	return challenge, nil
}

func (s *Service) Submit(ctx context.Context, submission Submission) (*Result, error) {
	// Idempotency: return the existing result if the user already submitted today.
	existing, err := s.repo.GetPlayerResult(ctx, submission.UserID, submission.ChallengeDate)
	if err == nil && existing != nil {
		existing.Percentile, _ = s.leaderboard.Percentile(ctx, submission.ChallengeDate, submission.UserID)
		return existing, nil
	}

	challenge, err := s.GetChallenge(ctx, submission.ChallengeDate)
	if err != nil {
		return nil, err
	}

	score := 0
	correctCount := 0
	for _, question := range challenge.Questions {
		if match.IsCorrectAnswer(submission.Answers[question.ID.String()], question.CorrectAnswers) {
			score += 100
			correctCount++
		}
	}

	if err := s.leaderboard.AddScore(ctx, submission.ChallengeDate, submission.UserID, score); err != nil {
		return nil, fmt.Errorf("record daily score: %w", err)
	}
	percentile, err := s.leaderboard.Percentile(ctx, submission.ChallengeDate, submission.UserID)
	if err != nil {
		percentile = 100
	}

	streak, err := s.repo.GetUserStreak(ctx, submission.UserID)
	if err != nil {
		return nil, fmt.Errorf("load streak: %w", err)
	}
	if streak == nil {
		streak = &Streak{}
	}
	freezeAvailable := 0
	if n, err := s.repo.GetStreakFreezeAvailable(ctx, submission.UserID); err == nil {
		freezeAvailable = n
	}
	useFreeze := false
	if streak.LastDate != nil {
		graceDur := time.Duration(s.graceHours) * time.Hour
		lastTrunc := streak.LastDate.UTC().Truncate(24 * time.Hour)
		diff := submission.ChallengeDate.UTC().Truncate(24*time.Hour).Sub(lastTrunc)
		useFreeze = diff > graceDur && diff <= graceDur+24*time.Hour && freezeAvailable > 0
	}
	updatedStreak := ComputeNextStreakWithFreeze(*streak, submission.ChallengeDate.UTC(), s.graceHours, freezeAvailable, useFreeze)
	if useFreeze {
		_ = s.repo.ConsumeStreakFreeze(ctx, submission.UserID)
	}
	if err := s.repo.UpdateUserStreak(ctx, submission.UserID, updatedStreak); err != nil {
		return nil, fmt.Errorf("update streak: %w", err)
	}

	result := Result{UserID: submission.UserID, ChallengeDate: submission.ChallengeDate.UTC(), Score: score, Percentile: percentile, StreakDay: updatedStreak.Current, CompletedAt: time.Now().UTC()}
	result.ShareCardText = s.GenerateShareCard(result, correctCount, len(challenge.Questions))
	if err := s.repo.SavePlayerResult(ctx, result); err != nil {
		return nil, fmt.Errorf("save daily result: %w", err)
	}
	return &result, nil
}

func (s *Service) GenerateShareCard(result Result, correctCount, totalQuestions int) string {
	cells := make([]string, 0, totalQuestions)
	for idx := 0; idx < totalQuestions; idx++ {
		if idx < correctCount {
			cells = append(cells, "G")
		} else {
			cells = append(cells, "R")
		}
	}
	return fmt.Sprintf(
		"ArchBattle Daily #%s\n%s\n%d/%d -- Top %.0f%% today\narchbattle.io",
		result.ChallengeDate.Format("20060102"),
		strings.Join(cells, ""),
		result.Score,
		totalQuestions*100,
		result.Percentile,
	)
}

func (s *Service) BufferDays(ctx context.Context, fromDate time.Time) (int, error) {
	return s.repo.BufferDays(ctx, fromDate)
}

func (s *Service) UpdateAISummary(ctx context.Context, date time.Time, summary string) error {
	return s.repo.UpdateAISummary(ctx, date, summary)
}

func NewResult(userID uuid.UUID, date time.Time) Result {
	return Result{UserID: userID, ChallengeDate: date.UTC()}
}
