package question

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type Service struct {
	repo             Repository
	disputeThreshold float64
}

func NewService(repo Repository, disputeThreshold float64) *Service {
	if disputeThreshold <= 0 {
		disputeThreshold = 0.08
	}
	return &Service{repo: repo, disputeThreshold: disputeThreshold}
}

func (s *Service) SelectQuestion(ctx context.Context, seenBy []uuid.UUID, tier shared.Tier, topic shared.Topic, mode shared.Mode, exclude []uuid.UUID, excludePilot bool) (*shared.QuestionSnapshot, error) {
	question, err := s.repo.SelectQuestion(ctx, seenBy, tier, topic, mode, time.Now().UTC().AddDate(0, 0, -7), exclude, excludePilot)
	if err == nil && question != nil {
		return toSnapshot(question), nil
	}
	question, err = s.repo.SelectQuestion(ctx, seenBy, tier, topic, mode, time.Now().UTC().AddDate(0, 0, -3), exclude, excludePilot)
	if err != nil {
		return nil, fmt.Errorf("select fallback question: %w", err)
	}
	if question == nil {
		return nil, fmt.Errorf("no question available")
	}
	return toSnapshot(question), nil
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
