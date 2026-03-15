package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domainquestion "github.com/radhakrishna/archbattle/core/internal/domain/question"
	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type QuestionRepo struct {
	pool *pgxpool.Pool
}

func NewQuestionRepo(pool *pgxpool.Pool) *QuestionRepo {
	return &QuestionRepo{pool: pool}
}

func (r *QuestionRepo) Create(ctx context.Context, question *domainquestion.Question) error {
	options, err := marshalJSON(question.Options)
	if err != nil {
		return err
	}
	correct, err := marshalJSON(question.CorrectAnswers)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
        INSERT INTO questions (id, mode, topic, difficulty_tier, prompt, options, correct_answers, rationale, dispute_count, pilot_attempts, pilot_dispute_rate, is_active, daily_eligible, reviewed_by, second_reviewer, status, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
    `, question.ID, question.Mode, question.Topic, question.DifficultyTier, question.Prompt, options, correct, question.Rationale, question.DisputeCount, question.PilotAttempts, question.PilotDisputeRate, question.IsActive, question.DailyEligible, question.ReviewedBy, question.SecondReviewer, question.Status, question.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert question: %w", err)
	}
	return nil
}

func (r *QuestionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domainquestion.Question, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, mode, topic, difficulty_tier, prompt, options, correct_answers, rationale, dispute_count, pilot_attempts, pilot_dispute_rate, is_active, daily_eligible, reviewed_by, second_reviewer, status, created_at
        FROM questions WHERE id = $1
    `, id)
	return scanFullQuestion(row)
}

func (r *QuestionRepo) ListByStatus(ctx context.Context, status string) ([]domainquestion.Question, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, mode, topic, difficulty_tier, prompt, options, correct_answers, rationale, dispute_count, pilot_attempts, pilot_dispute_rate, is_active, daily_eligible, reviewed_by, second_reviewer, status, created_at
        FROM questions WHERE status = $1 ORDER BY created_at DESC
    `, status)
	if err != nil {
		return nil, fmt.Errorf("query questions by status: %w", err)
	}
	defer rows.Close()

	questions := []domainquestion.Question{}
	for rows.Next() {
		question, err := scanFullQuestion(rows)
		if err != nil {
			return nil, err
		}
		questions = append(questions, *question)
	}
	return questions, rows.Err()
}

func (r *QuestionRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE questions
        SET status = $2,
            is_active = CASE WHEN $2 IN ('retired', 'quarantined') THEN FALSE ELSE is_active END
        WHERE id = $1
    `, id, status)
	if err != nil {
		return fmt.Errorf("update question status: %w", err)
	}
	return nil
}

func (r *QuestionRepo) UpdateRationale(ctx context.Context, id uuid.UUID, rationale string) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE questions
        SET rationale = $2, dispute_count = 0, status = 'live', is_active = TRUE
        WHERE id = $1
    `, id, rationale)
	if err != nil {
		return fmt.Errorf("update rationale: %w", err)
	}
	return nil
}

func (r *QuestionRepo) SelectQuestion(ctx context.Context, seenBy []uuid.UUID, tier shared.Tier, topic shared.Topic, mode shared.Mode, since time.Time, exclude []uuid.UUID, excludePilot bool) (*domainquestion.Question, error) {
	query := `
        SELECT q.id, q.mode, q.topic, q.difficulty_tier, q.prompt, q.options, q.correct_answers, q.rationale, q.dispute_count, q.pilot_attempts, q.pilot_dispute_rate, q.is_active, q.daily_eligible, q.reviewed_by, q.second_reviewer, q.status, q.created_at
        FROM questions q
        WHERE q.is_active = TRUE
          AND q.status IN ('live', 'pilot', 'staged')
          AND q.difficulty_tier = $1
          AND q.topic = $2
          AND q.mode = $3
    `
	if excludePilot {
		query += ` AND q.status != 'pilot' `
	}
	args := []any{tier, topic, mode}
	idx := 4

	if len(seenBy) > 0 {
		query += fmt.Sprintf(`
          AND q.id NOT IN (
                SELECT DISTINCT question_id
                FROM answer_submissions
                WHERE user_id = ANY($%d) AND created_at >= $%d
          )
        `, idx, idx+1)
		args = append(args, seenBy, since)
		idx += 2
	}
	if len(exclude) > 0 {
		query += fmt.Sprintf(" AND NOT (q.id = ANY($%d)) ", idx)
		args = append(args, exclude)
		idx++
	}

	query += ` ORDER BY RANDOM() LIMIT 1`
	row := r.pool.QueryRow(ctx, query, args...)
	return scanFullQuestion(row)
}

func (r *QuestionRepo) IncrementDispute(ctx context.Context, id uuid.UUID) (*domainquestion.Question, error) {
	_, err := r.pool.Exec(ctx, `UPDATE questions SET dispute_count = dispute_count + 1 WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("increment dispute count: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *QuestionRepo) IncrementPilotAttempt(ctx context.Context, id uuid.UUID) (*domainquestion.Question, error) {
	_, err := r.pool.Exec(ctx, `
        UPDATE questions
        SET pilot_attempts = pilot_attempts + 1,
            pilot_dispute_rate = CASE WHEN (pilot_attempts + 1) > 0
                THEN dispute_count::float / (pilot_attempts + 1) ELSE 0 END
        WHERE id = $1 AND status = 'pilot'
    `, id)
	if err != nil {
		return nil, fmt.Errorf("increment pilot attempt: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *QuestionRepo) ListUpcomingDaily(ctx context.Context, fromDate time.Time, days int) ([]domainquestion.Question, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, mode, topic, difficulty_tier, prompt, options, correct_answers, rationale, dispute_count, pilot_attempts, pilot_dispute_rate, is_active, daily_eligible, reviewed_by, second_reviewer, status, created_at
        FROM questions
        WHERE daily_eligible = TRUE AND is_active = TRUE
        ORDER BY created_at ASC
        LIMIT $1
    `, days*3)
	if err != nil {
		return nil, fmt.Errorf("query upcoming daily questions: %w", err)
	}
	defer rows.Close()

	questions := []domainquestion.Question{}
	for rows.Next() {
		question, err := scanFullQuestion(rows)
		if err != nil {
			return nil, err
		}
		questions = append(questions, *question)
	}
	return questions, rows.Err()
}

func scanFullQuestion(scanner rowScanner) (*domainquestion.Question, error) {
	question := &domainquestion.Question{}
	var options []byte
	var correct []byte
	if err := scanner.Scan(&question.ID, &question.Mode, &question.Topic, &question.DifficultyTier, &question.Prompt, &options, &correct, &question.Rationale, &question.DisputeCount, &question.PilotAttempts, &question.PilotDisputeRate, &question.IsActive, &question.DailyEligible, &question.ReviewedBy, &question.SecondReviewer, &question.Status, &question.CreatedAt); err != nil {
		return nil, err
	}
	if len(options) > 0 {
		if err := json.Unmarshal(options, &question.Options); err != nil {
			return nil, fmt.Errorf("decode question options: %w", err)
		}
	}
	if len(correct) > 0 {
		if err := json.Unmarshal(correct, &question.CorrectAnswers); err != nil {
			return nil, fmt.Errorf("decode correct answers: %w", err)
		}
	}
	question.Status = strings.ToLower(question.Status)
	return question, nil
}
