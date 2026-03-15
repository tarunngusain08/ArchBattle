package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domainmatch "github.com/radhakrishna/archbattle/core/internal/domain/match"
)

type MatchRepo struct {
	pool *pgxpool.Pool
}

func NewMatchRepo(pool *pgxpool.Pool) *MatchRepo {
	return &MatchRepo{pool: pool}
}

func (r *MatchRepo) Create(ctx context.Context, match *domainmatch.Match) error {
	_, err := r.pool.Exec(ctx, `
        INSERT INTO matches (id, mode, topic, tier, status, question_ids, started_at, ended_at, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `, match.ID, match.Mode, match.Topic, match.Tier, match.Status, match.QuestionIDs, match.StartedAt, match.EndedAt, match.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert match: %w", err)
	}
	return nil
}

func (r *MatchRepo) FindByID(ctx context.Context, id uuid.UUID) (*domainmatch.Match, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, mode, topic, tier, status, question_ids, started_at, ended_at, created_at
        FROM matches WHERE id = $1
    `, id)

	match := &domainmatch.Match{}
	if err := row.Scan(&match.ID, &match.Mode, &match.Topic, &match.Tier, &match.Status, &match.QuestionIDs, &match.StartedAt, &match.EndedAt, &match.CreatedAt); err != nil {
		return nil, err
	}
	match.QuestionIDs = parseUUIDArray(match.QuestionIDs)
	return match, nil
}

func (r *MatchRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domainmatch.MatchState) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE matches
        SET status = $2,
            started_at = CASE WHEN $2 = 'active' AND started_at IS NULL THEN NOW() ELSE started_at END,
            ended_at = CASE WHEN $2 IN ('ended', 'abandoned') THEN NOW() ELSE ended_at END
        WHERE id = $1
    `, id, status)
	if err != nil {
		return fmt.Errorf("update match status: %w", err)
	}
	return nil
}

func (r *MatchRepo) UpdateQuestionIDs(ctx context.Context, id uuid.UUID, questionIDs []uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE matches SET question_ids = $2 WHERE id = $1`, id, questionIDs)
	if err != nil {
		return fmt.Errorf("update question ids: %w", err)
	}
	return nil
}

func (r *MatchRepo) AddPlayers(ctx context.Context, players []domainmatch.MatchPlayer) error {
	for _, player := range players {
		answers, err := marshalJSON(player.Answers)
		if err != nil {
			return err
		}
		_, err = r.pool.Exec(ctx, `
            INSERT INTO match_players (match_id, user_id, username, score, answers, elo_before, elo_after, elo_delta, disconnected, joined_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
            ON CONFLICT (match_id, user_id) DO NOTHING
        `, player.MatchID, player.UserID, player.Username, player.Score, answers, player.ELOBefore, player.ELOAfter, player.ELODelta, player.Disconnected, player.JoinedAt)
		if err != nil {
			return fmt.Errorf("insert match player: %w", err)
		}
	}
	return nil
}

func (r *MatchRepo) GetPlayers(ctx context.Context, matchID uuid.UUID) ([]domainmatch.MatchPlayer, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT match_id, user_id, username, score, answers, elo_before, elo_after, elo_delta, disconnected, joined_at
        FROM match_players WHERE match_id = $1
        ORDER BY joined_at ASC
    `, matchID)
	if err != nil {
		return nil, fmt.Errorf("query match players: %w", err)
	}
	defer rows.Close()

	players := []domainmatch.MatchPlayer{}
	for rows.Next() {
		var player domainmatch.MatchPlayer
		var answers []byte
		if err := rows.Scan(&player.MatchID, &player.UserID, &player.Username, &player.Score, &answers, &player.ELOBefore, &player.ELOAfter, &player.ELODelta, &player.Disconnected, &player.JoinedAt); err != nil {
			return nil, fmt.Errorf("scan match player: %w", err)
		}
		if len(answers) > 0 {
			if err := json.Unmarshal(answers, &player.Answers); err != nil {
				return nil, fmt.Errorf("decode player answers: %w", err)
			}
		}
		if player.Answers == nil {
			player.Answers = map[string][]int{}
		}
		players = append(players, player)
	}
	return players, rows.Err()
}

func (r *MatchRepo) UpdatePlayerResult(ctx context.Context, matchID uuid.UUID, standing domainmatch.PlayerStanding) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE match_players
        SET score = $3,
            elo_after = $4,
            elo_delta = $5,
            disconnected = $6
        WHERE match_id = $1 AND user_id = $2
    `, matchID, standing.UserID, standing.Score, standing.ELOAfter, standing.ELODelta, standing.Disconnected)
	if err != nil {
		return fmt.Errorf("update player result: %w", err)
	}
	return nil
}

// MatchHistoryEntry is a row for the user's match history.
type MatchHistoryEntry struct {
	Opponent string
	Score    int
	ELODelta int
}

func (r *MatchRepo) UserMatchHistory(ctx context.Context, userID uuid.UUID, limit int) ([]MatchHistoryEntry, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.pool.Query(ctx, `
        WITH user_matches AS (
            SELECT mp.match_id, mp.score, mp.elo_delta
            FROM match_players mp
            JOIN matches m ON m.id = mp.match_id AND m.status = 'ended'
            WHERE mp.user_id = $1
            ORDER BY m.ended_at DESC NULLS LAST, m.created_at DESC
            LIMIT $2
        )
        SELECT COALESCE(
            (SELECT username FROM match_players WHERE match_id = um.match_id AND user_id != $1 LIMIT 1),
            'Solo'
        ) AS opponent, um.score, um.elo_delta
        FROM user_matches um
    `, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("query match history: %w", err)
	}
	defer rows.Close()

	var entries []MatchHistoryEntry
	for rows.Next() {
		var e MatchHistoryEntry
		if err := rows.Scan(&e.Opponent, &e.Score, &e.ELODelta); err != nil {
			return nil, fmt.Errorf("scan match history: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// TopicStat holds accuracy for a topic.
type TopicStat struct {
	Topic   string
	Correct int
	Total   int
}

func (r *MatchRepo) UserTopicStats(ctx context.Context, userID uuid.UUID) ([]TopicStat, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT q.topic, COUNT(*) FILTER (WHERE s.is_correct) AS correct, COUNT(*) AS total
        FROM answer_submissions s
        JOIN questions q ON q.id = s.question_id
        WHERE s.user_id = $1
        GROUP BY q.topic
    `, userID)
	if err != nil {
		return nil, fmt.Errorf("query topic stats: %w", err)
	}
	defer rows.Close()

	var stats []TopicStat
	for rows.Next() {
		var s TopicStat
		if err := rows.Scan(&s.Topic, &s.Correct, &s.Total); err != nil {
			return nil, fmt.Errorf("scan topic stat: %w", err)
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}
