package repository

import (
	"context"
	"time"

	"user-analytics-service/internal/domain"
)

// Repository is the persistence boundary for user activity data.
type Repository interface {
	// SaveLoginEvent persists a login event. Duplicate (user_id, timestamp)
	// pairs are treated as idempotent no-ops.
	SaveLoginEvent(ctx context.Context, event domain.LoginEvent) error

	// CountDistinctUsers returns the count of distinct user IDs with an
	// activity timestamp in [start, end).
	CountDistinctUsers(ctx context.Context, start, end time.Time) (int64, error)
}
