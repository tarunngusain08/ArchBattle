package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domaindaily "github.com/radhakrishna/archbattle/core/internal/domain/daily"
	domainleaderboard "github.com/radhakrishna/archbattle/core/internal/domain/leaderboard"
	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type DailyRepo struct {
	pool *pgxpool.Pool
}

func NewDailyRepo(pool *pgxpool.Pool) *DailyRepo {
	return &DailyRepo{pool: pool}
}

func (r *DailyRepo) GetByDate(ctx context.Context, date time.Time) (*domaindaily.DailyChallenge, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, challenge_date, question_ids, theme, ai_summary, summary_reviewed, published_at
        FROM daily_challenges WHERE challenge_date = $1
    `, normalizeDate(date))

	challenge := &domaindaily.DailyChallenge{}
	if err := row.Scan(&challenge.ID, &challenge.ChallengeDate, &challenge.QuestionIDs, &challenge.Theme, &challenge.AISummary, &challenge.SummaryReviewed, &challenge.PublishedAt); err != nil {
		return nil, err
	}
	challenge.QuestionIDs = parseUUIDArray(challenge.QuestionIDs)

	if len(challenge.QuestionIDs) > 0 {
		rows, err := r.pool.Query(ctx, `
            SELECT id, mode, topic, difficulty_tier, prompt, options, correct_answers, rationale
            FROM questions WHERE id = ANY($1)
        `, challenge.QuestionIDs)
		if err != nil {
			return nil, fmt.Errorf("query challenge questions: %w", err)
		}
		defer rows.Close()

		questionMap := map[uuid.UUID]shared.QuestionSnapshot{}
		for rows.Next() {
			snapshot, err := scanQuestion(rows)
			if err != nil {
				return nil, err
			}
			questionMap[snapshot.ID] = *snapshot
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		challenge.Questions = make([]shared.QuestionSnapshot, 0, len(challenge.QuestionIDs))
		for _, questionID := range challenge.QuestionIDs {
			if snapshot, ok := questionMap[questionID]; ok {
				challenge.Questions = append(challenge.Questions, snapshot)
			}
		}
	}
	return challenge, nil
}

func (r *DailyRepo) SavePlayerResult(ctx context.Context, result domaindaily.Result) error {
	_, err := r.pool.Exec(ctx, `
        INSERT INTO player_daily_challenges (user_id, challenge_date, score, streak_day, share_card_text, completed_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (user_id, challenge_date)
        DO UPDATE SET score = EXCLUDED.score,
                      streak_day = EXCLUDED.streak_day,
                      share_card_text = EXCLUDED.share_card_text,
                      completed_at = EXCLUDED.completed_at
    `, result.UserID, normalizeDate(result.ChallengeDate), result.Score, result.StreakDay, result.ShareCardText, result.CompletedAt)
	if err != nil {
		return fmt.Errorf("upsert player daily challenge: %w", err)
	}
	return nil
}

func (r *DailyRepo) GetPlayerResult(ctx context.Context, userID uuid.UUID, date time.Time) (*domaindaily.Result, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT user_id, challenge_date, score, streak_day, share_card_text, completed_at
        FROM player_daily_challenges
        WHERE user_id = $1 AND challenge_date = $2
    `, userID, normalizeDate(date))
	result := &domaindaily.Result{}
	if err := row.Scan(&result.UserID, &result.ChallengeDate, &result.Score, &result.StreakDay, &result.ShareCardText, &result.CompletedAt); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *DailyRepo) GetUserStreak(ctx context.Context, userID uuid.UUID) (*domaindaily.Streak, error) {
	row := r.pool.QueryRow(ctx, `SELECT current_streak, longest_streak, last_daily_date FROM users WHERE id = $1`, userID)
	streak := &domaindaily.Streak{}
	if err := row.Scan(&streak.Current, &streak.Longest, &streak.LastDate); err != nil {
		return nil, err
	}
	return streak, nil
}

func (r *DailyRepo) UpdateUserStreak(ctx context.Context, userID uuid.UUID, streak domaindaily.Streak) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE users
        SET current_streak = $2,
            longest_streak = $3,
            last_daily_date = $4
        WHERE id = $1
    `, userID, streak.Current, streak.Longest, streak.LastDate)
	if err != nil {
		return fmt.Errorf("update user streak: %w", err)
	}
	return nil
}

func (r *DailyRepo) BufferDays(ctx context.Context, fromDate time.Time) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM daily_challenges WHERE challenge_date >= $1`, normalizeDate(fromDate)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count upcoming daily challenges: %w", err)
	}
	return count, nil
}

func (r *DailyRepo) UpdateAISummary(ctx context.Context, date time.Time, summary string) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE daily_challenges SET ai_summary = $2 WHERE challenge_date = $1
    `, normalizeDate(date), summary)
	if err != nil {
		return fmt.Errorf("update ai summary: %w", err)
	}
	return nil
}

func (r *DailyRepo) GetStreakFreezeAvailable(ctx context.Context, userID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT streak_freeze_available FROM users WHERE id = $1`, userID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("get streak freeze: %w", err)
	}
	return n, nil
}

func (r *DailyRepo) ConsumeStreakFreeze(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE users SET streak_freeze_available = GREATEST(0, streak_freeze_available - 1) WHERE id = $1
    `, userID)
	if err != nil {
		return fmt.Errorf("consume streak freeze: %w", err)
	}
	return nil
}

func (r *DailyRepo) CreditWeeklyFreeze(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET streak_freeze_available = 1`)
	if err != nil {
		return fmt.Errorf("credit weekly freeze: %w", err)
	}
	return nil
}

func (r *DailyRepo) RecentDailyDates(ctx context.Context, userID uuid.UUID, days int) ([]time.Time, error) {
	if days <= 0 {
		days = 30
	}
	rows, err := r.pool.Query(ctx, `
        SELECT challenge_date FROM player_daily_challenges
        WHERE user_id = $1 AND challenge_date >= $2
        ORDER BY challenge_date DESC
        LIMIT $3
    `, userID, normalizeDate(time.Now().UTC().AddDate(0, 0, -days)), days)
	if err != nil {
		return nil, fmt.Errorf("query recent daily dates: %w", err)
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			return nil, fmt.Errorf("scan date: %w", err)
		}
		dates = append(dates, d.UTC().Truncate(24*time.Hour))
	}
	return dates, rows.Err()
}

func (r *DailyRepo) Top(ctx context.Context, date time.Time, limit int) ([]domainleaderboard.Entry, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT user_id, score FROM player_daily_challenges
        WHERE challenge_date = $1
        ORDER BY score DESC, completed_at ASC
        LIMIT $2
    `, normalizeDate(date), limit)
	if err != nil {
		return nil, fmt.Errorf("query daily top: %w", err)
	}
	defer rows.Close()

	entries := []domainleaderboard.Entry{}
	rank := int64(1)
	for rows.Next() {
		entry := domainleaderboard.Entry{Scope: domainleaderboard.ScopeGlobal}
		if err := rows.Scan(&entry.UserID, &entry.Score); err != nil {
			return nil, fmt.Errorf("scan daily leaderboard entry: %w", err)
		}
		entry.Rank = rank
		rank++
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}
