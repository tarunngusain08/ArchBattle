package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domainmatch "github.com/radhakrishna/archbattle/core/internal/domain/match"
)

type SubmissionRepo struct {
	pool *pgxpool.Pool
}

func NewSubmissionRepo(pool *pgxpool.Pool) *SubmissionRepo {
	return &SubmissionRepo{pool: pool}
}

func (r *SubmissionRepo) SaveSubmission(ctx context.Context, sub *domainmatch.AnswerSubmission) error {
	payload, err := marshalJSON(sub.ChosenOptions)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
        INSERT INTO answer_submissions (id, match_id, user_id, question_id, chosen_options, is_correct, points_awarded, server_received_at, elapsed_seconds)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `, sub.ID, sub.MatchID, sub.UserID, sub.QuestionID, payload, sub.IsCorrect, sub.PointsAwarded, sub.ServerReceivedAt, sub.ElapsedSeconds)
	if err != nil {
		return fmt.Errorf("insert submission: %w", err)
	}
	return nil
}

func (r *SubmissionRepo) ListByMatch(ctx context.Context, matchID uuid.UUID) ([]domainmatch.AnswerSubmission, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, match_id, user_id, question_id, chosen_options, is_correct, points_awarded, server_received_at, elapsed_seconds
        FROM answer_submissions WHERE match_id = $1
        ORDER BY server_received_at ASC
    `, matchID)
	if err != nil {
		return nil, fmt.Errorf("query submissions: %w", err)
	}
	defer rows.Close()

	submissions := []domainmatch.AnswerSubmission{}
	for rows.Next() {
		var sub domainmatch.AnswerSubmission
		var choices []byte
		if err := rows.Scan(&sub.ID, &sub.MatchID, &sub.UserID, &sub.QuestionID, &choices, &sub.IsCorrect, &sub.PointsAwarded, &sub.ServerReceivedAt, &sub.ElapsedSeconds); err != nil {
			return nil, fmt.Errorf("scan submission: %w", err)
		}
		if err := json.Unmarshal(choices, &sub.ChosenOptions); err != nil {
			return nil, fmt.Errorf("decode chosen options: %w", err)
		}
		submissions = append(submissions, sub)
	}
	return submissions, rows.Err()
}
