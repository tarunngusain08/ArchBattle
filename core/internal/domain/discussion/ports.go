package discussion

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the persistence contract for discussion entries.
type Repository interface {
	Create(ctx context.Context, req CreateRequest) (*Entry, error)
	ListByDate(ctx context.Context, date time.Time) ([]*Entry, error)
	Upvote(ctx context.Context, entryID, voterID uuid.UUID) error
}
