package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	domainshared "github.com/radhakrishna/archbattle/ai-service/internal/domain/shared"
)

// SessionLogger persists AI tutor session records to the ai_tutor_sessions table.
// Fields match_id and question_id are stored as NULL when not supplied, so the
// table migration must allow nullable UUIDs for those columns.
type SessionLogger struct {
	pool *pgxpool.Pool
}

// NewSessionLogger creates a Postgres-backed SessionLogger.
func NewSessionLogger(pool *pgxpool.Pool) *SessionLogger {
	return &SessionLogger{pool: pool}
}

// Log inserts one session record into ai_tutor_sessions.
func (l *SessionLogger) Log(ctx context.Context, record domainshared.SessionRecord) error {
	messagesJSON, err := json.Marshal(record.Messages)
	if err != nil {
		return fmt.Errorf("marshal messages: %w", err)
	}

	var matchID, questionID *string
	if record.MatchID != "" {
		matchID = &record.MatchID
	}
	if record.QuestionID != "" {
		questionID = &record.QuestionID
	}

	_, err = l.pool.Exec(ctx, `
		INSERT INTO ai_tutor_sessions (user_id, match_id, question_id, messages, token_count, created_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::jsonb, $5, $6)
	`, record.UserID, matchID, questionID, messagesJSON, record.TokenCount, record.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert tutor session: %w", err)
	}
	return nil
}
