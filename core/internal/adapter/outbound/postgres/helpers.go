package postgres

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func marshalJSON(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}
	return data, nil
}

func scanQuestion(scanner rowScanner) (*shared.QuestionSnapshot, error) {
	var snapshot shared.QuestionSnapshot
	var options []byte
	var correctAnswers []byte

	if err := scanner.Scan(&snapshot.ID, &snapshot.Mode, &snapshot.Topic, &snapshot.DifficultyTier, &snapshot.Prompt, &options, &correctAnswers, &snapshot.Rationale); err != nil {
		return nil, err
	}
	if len(options) > 0 {
		if err := json.Unmarshal(options, &snapshot.Options); err != nil {
			return nil, fmt.Errorf("decode options: %w", err)
		}
	}
	if snapshot.Options == nil {
		snapshot.Options = []string{}
	}
	if len(correctAnswers) > 0 {
		if err := json.Unmarshal(correctAnswers, &snapshot.CorrectAnswers); err != nil {
			return nil, fmt.Errorf("decode correct answers: %w", err)
		}
	}
	if snapshot.CorrectAnswers == nil {
		snapshot.CorrectAnswers = []int{}
	}
	return &snapshot, nil
}

func parseUUIDArray(text []uuid.UUID) []uuid.UUID {
	if text == nil {
		return []uuid.UUID{}
	}
	return text
}

func normalizeDate(value time.Time) time.Time {
	return value.UTC().Truncate(24 * time.Hour)
}

func isNoRows(err error) bool {
	return err == pgx.ErrNoRows
}
