package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domaindiscussion "github.com/radhakrishna/archbattle/core/internal/domain/discussion"
)

type DiscussionRepo struct {
	pool *pgxpool.Pool
}

func NewDiscussionRepo(pool *pgxpool.Pool) *DiscussionRepo {
	return &DiscussionRepo{pool: pool}
}

func (r *DiscussionRepo) Create(ctx context.Context, req domaindiscussion.CreateRequest) (*domaindiscussion.Entry, error) {
	date := normalizeDate(req.ChallengeDate)
	row := r.pool.QueryRow(ctx, `
        INSERT INTO discussion_entries (challenge_date, user_id, question_number, reasoning_text, alternative_text, surprise_text)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, challenge_date, user_id, question_number, reasoning_text, alternative_text, surprise_text, upvotes, created_at
    `, date, req.UserID, req.QuestionNumber, req.ReasoningText, req.AlternativeText, req.SurpriseText)

	var entry domaindiscussion.Entry
	if err := row.Scan(&entry.ID, &entry.ChallengeDate, &entry.UserID, &entry.QuestionNumber, &entry.ReasoningText, &entry.AlternativeText, &entry.SurpriseText, &entry.Upvotes, &entry.CreatedAt); err != nil {
		return nil, fmt.Errorf("insert discussion entry: %w", err)
	}
	entry.ChallengeDate = entry.ChallengeDate.UTC().Truncate(24 * time.Hour)

	usernameRow := r.pool.QueryRow(ctx, `SELECT username FROM users WHERE id = $1`, req.UserID)
	_ = usernameRow.Scan(&entry.Username)
	return &entry, nil
}

func (r *DiscussionRepo) ListByDate(ctx context.Context, date time.Time) ([]*domaindiscussion.Entry, error) {
	normDate := normalizeDate(date)
	rows, err := r.pool.Query(ctx, `
        SELECT e.id, e.challenge_date, e.user_id, u.username, e.question_number,
               e.reasoning_text, e.alternative_text, e.surprise_text, e.upvotes, e.created_at
        FROM discussion_entries e
        JOIN users u ON u.id = e.user_id
        WHERE e.challenge_date = $1
        ORDER BY e.upvotes DESC, e.created_at ASC
    `, normDate)
	if err != nil {
		return nil, fmt.Errorf("list discussion entries: %w", err)
	}
	defer rows.Close()

	var entries []*domaindiscussion.Entry
	for rows.Next() {
		var e domaindiscussion.Entry
		if err := rows.Scan(&e.ID, &e.ChallengeDate, &e.UserID, &e.Username, &e.QuestionNumber, &e.ReasoningText, &e.AlternativeText, &e.SurpriseText, &e.Upvotes, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		entries = append(entries, &e)
	}
	return entries, rows.Err()
}

func (r *DiscussionRepo) Upvote(ctx context.Context, entryID, voterID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
        INSERT INTO discussion_upvotes (entry_id, voter_id) VALUES ($1, $2)
        ON CONFLICT (entry_id, voter_id) DO NOTHING
    `, entryID, voterID)
	if err != nil {
		return fmt.Errorf("insert upvote: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	_, err = r.pool.Exec(ctx, `UPDATE discussion_entries SET upvotes = upvotes + 1 WHERE id = $1`, entryID)
	if err != nil {
		return fmt.Errorf("increment upvotes: %w", err)
	}
	return nil
}
