package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"user-analytics-service/internal/domain"
)

// PostgresRepository is a pgx-backed implementation of Repository.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository builds a PostgresRepository backed by pool.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

var _ Repository = (*PostgresRepository)(nil)

func (r *PostgresRepository) SaveLoginEvent(ctx context.Context, event domain.LoginEvent) error {
	const query = `
		INSERT INTO ` + userActivityLogTable + ` (user_id, timestamp)
		VALUES ($1, $2)
		ON CONFLICT (user_id, timestamp) DO NOTHING`

	_, err := r.pool.Exec(ctx, query, event.UserID, event.Timestamp)
	return err
}

func (r *PostgresRepository) CountDistinctUsers(ctx context.Context, start, end time.Time) (int64, error) {
	const query = `
		SELECT COUNT(DISTINCT user_id)
		FROM ` + userActivityLogTable + `
		WHERE timestamp >= $1 AND timestamp < $2`

	var count int64
	if err := r.pool.QueryRow(ctx, query, start, end).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
