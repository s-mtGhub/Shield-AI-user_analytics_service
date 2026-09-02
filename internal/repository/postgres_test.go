package repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	"user-analytics-service/internal/domain"
	"user-analytics-service/internal/repository"
)

// TestPostgresRepository is an integration test against a real Postgres
// instance. It's skipped unless TEST_DATABASE_URL is set, e.g.:
//
//	TEST_DATABASE_URL=postgres://user:pass@localhost:5432/analytics_test?sslmode=disable \
//	  go test ./internal/repository/...
func TestPostgresRepository(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping Postgres integration test")
	}

	m, err := migrate.New("file://../../migrations", dbURL)
	if err != nil {
		t.Fatalf("failed to init migrator: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("failed to run migrations: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "TRUNCATE TABLE user_activity_log")
	})

	repo := repository.NewPostgresRepository(pool)

	day := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	events := []domain.LoginEvent{
		{UserID: "user-1", Timestamp: day},
		{UserID: "user-1", Timestamp: day.Add(time.Hour)}, // same user, same day
		{UserID: "user-2", Timestamp: day.Add(2 * time.Hour)},
		{UserID: "user-3", Timestamp: day.AddDate(0, 0, 1)}, // next day
	}
	for _, e := range events {
		if err := repo.SaveLoginEvent(ctx, e); err != nil {
			t.Fatalf("failed to save event %+v: %v", e, err)
		}
	}

	// Re-saving the same (user_id, timestamp) must be idempotent.
	if err := repo.SaveLoginEvent(ctx, events[0]); err != nil {
		t.Fatalf("expected duplicate save to be a no-op, got error: %v", err)
	}

	start := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)
	count, err := repo.CountDistinctUsers(ctx, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 distinct users on 2026-08-31, got %d", count)
	}

	monthEnd := start.AddDate(0, 1, 0)
	count, err = repo.CountDistinctUsers(ctx, start, monthEnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 distinct users in August 2026, got %d", count)
	}
}
