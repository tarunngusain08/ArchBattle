package matchmaking

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type Service struct {
	queue   QueueStore
	factory MatchFactory
	notifier Notifier
	rematch RematchLoader
}

func NewService(queue QueueStore, factory MatchFactory, notifier Notifier, rematch RematchLoader) *Service {
	return &Service{queue: queue, factory: factory, notifier: notifier, rematch: rematch}
}

func (s *Service) Enqueue(ctx context.Context, entry QueueEntry) error {
	if entry.UserID == uuid.Nil {
		return fmt.Errorf("user id is required")
	}
	if entry.JoinedAt.IsZero() {
		entry.JoinedAt = time.Now().UTC()
	}
	return s.queue.Enqueue(ctx, entry)
}

func (s *Service) Dequeue(ctx context.Context, entry QueueEntry) error {
	return s.queue.Dequeue(ctx, string(entry.Tier), string(entry.Topic), entry.UserID)
}

func (s *Service) CreateSoloMatch(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	entry, err := s.queue.GetEntry(ctx, userID)
	if err != nil || entry == nil {
		return uuid.Nil, fmt.Errorf("queue entry not found: %w", err)
	}
	_ = s.queue.Dequeue(ctx, string(entry.Tier), string(entry.Topic), entry.UserID)
	matchID, err := s.factory.CreateMatch(ctx, MatchRequest{Players: []QueueEntry{*entry}, Tier: entry.Tier, Topic: entry.Topic, Mode: entry.Mode, Solo: true})
	if err != nil {
		return uuid.Nil, fmt.Errorf("create solo match: %w", err)
	}
	_ = s.notifier.NotifyMatchFound(ctx, entry.UserID, matchID)
	return matchID, nil
}

func (s *Service) AcceptCrossMatch(ctx context.Context, userID uuid.UUID, targetTier string) error {
	entry, err := s.queue.GetEntry(ctx, userID)
	if err != nil || entry == nil {
		return fmt.Errorf("queue entry not found: %w", err)
	}
	oldTier := string(entry.Tier)
	_ = s.queue.Dequeue(ctx, oldTier, string(entry.Topic), entry.UserID)
	entry.Tier = shared.Tier(targetTier)
	return s.queue.Enqueue(ctx, *entry)
}

func (s *Service) RequestRematch(ctx context.Context, matchID uuid.UUID) error {
	if s.rematch == nil {
		return fmt.Errorf("rematch not configured")
	}
	tier, topic, mode, players, err := s.rematch.LoadForRematch(ctx, matchID)
	if err != nil {
		return fmt.Errorf("load match for rematch: %w", err)
	}
	for _, p := range players {
		p.Tier = shared.Tier(tier)
		p.Topic = shared.Topic(topic)
		p.Mode = shared.Mode(mode)
		if err := s.queue.Enqueue(ctx, p); err != nil {
			return fmt.Errorf("enqueue player %s: %w", p.UserID, err)
		}
	}
	return nil
}

func (s *Service) Run(ctx context.Context) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if err := s.RunScanOnce(ctx); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Service) RunScanOnce(ctx context.Context) error {
	queues, err := s.queue.GetAllActiveQueues(ctx)
	if err != nil {
		return fmt.Errorf("list active queues: %w", err)
	}

	for _, key := range queues {
		parts := strings.Split(key, ":")
		if len(parts) != 3 {
			continue
		}
		tier := parts[1]
		topic := parts[2]

		candidates, err := s.queue.FindCandidates(ctx, tier, topic, 0, 4000)
		if err != nil {
			return fmt.Errorf("load candidates for %s: %w", key, err)
		}

		used := map[uuid.UUID]struct{}{}
		for _, entry := range candidates {
			if _, seen := used[entry.UserID]; seen {
				continue
			}

			joinedAt, err := s.queue.GetEntryTime(ctx, entry.UserID)
			if err != nil {
				continue
			}
			waited := time.Since(joinedAt)
			minELO, maxELO := entry.ELO-150, entry.ELO+150
			if waited >= 60*time.Second {
				minELO, maxELO = entry.ELO-300, entry.ELO+300
			}

			peers, err := s.queue.FindCandidates(ctx, tier, topic, minELO, maxELO)
			if err != nil {
				return fmt.Errorf("find peers for %s: %w", entry.UserID, err)
			}

			group := make([]QueueEntry, 0, 4)
			for _, peer := range peers {
				if len(group) == 4 {
					break
				}
				if _, seen := used[peer.UserID]; seen {
					continue
				}
				group = append(group, peer)
			}

			if len(group) >= 2 {
				matchID, err := s.factory.CreateMatch(ctx, MatchRequest{Players: group, Tier: entry.Tier, Topic: entry.Topic, Mode: entry.Mode})
				if err != nil {
					return fmt.Errorf("create match: %w", err)
				}
				for _, player := range group {
					used[player.UserID] = struct{}{}
					_ = s.queue.Dequeue(ctx, tier, topic, player.UserID)
					_ = s.notifier.NotifyMatchFound(ctx, player.UserID, matchID)
				}
				continue
			}

			if waited >= 90*time.Second {
				if entry.Tier == "staff" {
					_ = s.notifier.NotifyCrossMatch(ctx, entry.UserID, "senior")
				}
				_ = s.notifier.NotifySoloFallback(ctx, entry.UserID)
			}
		}
	}

	return nil
}
