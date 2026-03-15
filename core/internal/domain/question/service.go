package question

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type Service struct {
	repo             Repository
	disputeThreshold float64
	aiGen            AIQuestionGenerator
}

func NewService(repo Repository, disputeThreshold float64, aiGen AIQuestionGenerator) *Service {
	if disputeThreshold <= 0 {
		disputeThreshold = 0.08
	}
	return &Service{repo: repo, disputeThreshold: disputeThreshold, aiGen: aiGen}
}

func (s *Service) SelectQuestion(ctx context.Context, seenBy []uuid.UUID, tier shared.Tier, topic shared.Topic, mode shared.Mode, exclude []uuid.UUID, excludePilot bool) (*shared.QuestionSnapshot, error) {
	// 1. Try AI generation (with timeout)
	if s.aiGen != nil {
		aiCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if q, err := s.aiGen.GenerateQuestion(aiCtx, string(topic), string(tier), string(mode)); err == nil {
			q.ID = uuid.New()
			q.Status = "live"
			q.IsActive = true
			q.CreatedAt = time.Now().UTC()
			if err := s.repo.Create(ctx, q); err != nil {
				return nil, fmt.Errorf("persist ai question: %w", err)
			}
			return toSnapshot(q), nil
		} else {
			slog.Warn("ai question generation failed, using fallback", "topic", topic, "tier", tier, "error", err)
		}
	}
	// 2. Fallback: random from DB (topic-agnostic)
	question, err := s.repo.SelectFallbackRandom(ctx, tier, mode)
	if err != nil {
		return nil, fmt.Errorf("select fallback question: %w", err)
	}
	if question == nil {
		return nil, fmt.Errorf("no question available")
	}
	return toSnapshot(question), nil
}

func (s *Service) SelectQuestions(ctx context.Context, count int, seenBy []uuid.UUID, tier shared.Tier, topic shared.Topic, mode shared.Mode, exclude []uuid.UUID, excludePilot bool) ([]*shared.QuestionSnapshot, error) {
	result := make([]*shared.QuestionSnapshot, 0, count)
	aiQuestionIDs := make([]uuid.UUID, 0, count)

	if s.aiGen != nil {
		aiCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		aiQuestions, err := s.aiGen.GenerateQuestions(aiCtx, string(topic), string(tier), string(mode), count)
		if err != nil {
			slog.Warn("ai bulk question generation failed, using fallback", "topic", topic, "tier", tier, "error", err)
		} else {
			for _, q := range aiQuestions {
				q.ID = uuid.New()
				q.Status = "live"
				q.IsActive = true
				q.CreatedAt = time.Now().UTC()
				if err := s.repo.Create(ctx, q); err != nil {
					slog.Warn("persist ai question failed, skipping", "error", err)
					continue
				}
				result = append(result, toSnapshot(q))
				aiQuestionIDs = append(aiQuestionIDs, q.ID)
			}
		}
	}

	needed := count - len(result)
	if needed > 0 {
		excludeIDs := append([]uuid.UUID(nil), exclude...)
		excludeIDs = append(excludeIDs, aiQuestionIDs...)
		fallbacks, err := s.repo.SelectFallbackRandomN(ctx, tier, mode, needed, excludeIDs)
		if err != nil {
			return nil, fmt.Errorf("select fallback questions: %w", err)
		}
		for _, q := range fallbacks {
			result = append(result, toSnapshot(q))
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no questions available")
	}
	return result, nil
}

func (s *Service) IncrementPilotAttempt(ctx context.Context, id uuid.UUID) error {
	q, err := s.repo.GetByID(ctx, id)
	if err != nil || q == nil || q.Status != "pilot" {
		return nil
	}
	updated, err := s.repo.IncrementPilotAttempt(ctx, id)
	if err != nil {
		return fmt.Errorf("increment pilot attempt: %w", err)
	}
	if updated == nil {
		return nil
	}
	if updated.PilotAttempts >= 100 && updated.PilotDisputeRate < 0.05 {
		if err := s.repo.UpdateStatus(ctx, id, "live"); err != nil {
			return fmt.Errorf("auto-promote to live: %w", err)
		}
	} else if updated.PilotAttempts >= 50 && updated.PilotDisputeRate > 0.2 {
		if err := s.repo.UpdateStatus(ctx, id, "quarantined"); err != nil {
			return fmt.Errorf("auto-quarantine: %w", err)
		}
	}
	return nil
}

func (s *Service) SubmitDispute(ctx context.Context, dispute Dispute) (*Question, error) {
	updated, err := s.repo.IncrementDispute(ctx, dispute.QuestionID)
	if err != nil {
		return nil, fmt.Errorf("increment dispute: %w", err)
	}
	if updated == nil {
		return nil, fmt.Errorf("question not found")
	}
	if updated.PilotAttempts > 0 {
		updated.PilotDisputeRate = float64(updated.DisputeCount) / float64(updated.PilotAttempts)
		if updated.PilotDisputeRate > s.disputeThreshold {
			if err := s.repo.UpdateStatus(ctx, updated.ID, "quarantined"); err != nil {
				return nil, fmt.Errorf("auto quarantine question: %w", err)
			}
			updated.Status = "quarantined"
			updated.IsActive = false
		}
	}
	return updated, nil
}

func (s *Service) ListByStatus(ctx context.Context, status string) ([]Question, error) {
	return s.repo.ListByStatus(ctx, status)
}

func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return s.repo.UpdateStatus(ctx, id, status)
}

func (s *Service) UpdateRationale(ctx context.Context, id uuid.UUID, rationale string) error {
	return s.repo.UpdateRationale(ctx, id, rationale)
}

func (s *Service) Create(ctx context.Context, question *Question) error {
	return s.repo.Create(ctx, question)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Question, error) {
	question, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get question by id: %w", err)
	}
	if question == nil {
		return nil, fmt.Errorf("question not found")
	}
	return question, nil
}

func toSnapshot(question *Question) *shared.QuestionSnapshot {
	return &shared.QuestionSnapshot{ID: question.ID, Prompt: question.Prompt, Options: append([]string(nil), question.Options...), CorrectAnswers: append([]int(nil), question.CorrectAnswers...), Rationale: question.Rationale, Topic: question.Topic, DifficultyTier: question.DifficultyTier, Mode: question.Mode, Status: question.Status}
}
