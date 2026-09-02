package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"user-analytics-service/internal/service"
)

func TestQueryService_DailyActiveUsers(t *testing.T) {
	t.Run("resolves UTC day bounds and returns the count", func(t *testing.T) {
		var gotStart, gotEnd time.Time
		repo := &fakeRepository{
			countFunc: func(start, end time.Time) (int64, error) {
				gotStart, gotEnd = start, end
				return 42, nil
			},
		}
		svc := service.NewQueryService(repo, time.UTC)

		count, err := svc.DailyActiveUsers(context.Background(), "2026-08-31")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 42 {
			t.Fatalf("expected count 42, got %d", count)
		}

		wantStart := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
		wantEnd := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
		if !gotStart.Equal(wantStart) || !gotEnd.Equal(wantEnd) {
			t.Fatalf("expected bounds [%v, %v), got [%v, %v)", wantStart, wantEnd, gotStart, gotEnd)
		}
	})

	t.Run("resolves day bounds in a non-UTC service timezone", func(t *testing.T) {
		ist, err := time.LoadLocation("Asia/Kolkata")
		if err != nil {
			t.Fatalf("failed to load Asia/Kolkata: %v", err)
		}

		var gotStart, gotEnd time.Time
		repo := &fakeRepository{
			countFunc: func(start, end time.Time) (int64, error) {
				gotStart, gotEnd = start, end
				return 1, nil
			},
		}
		svc := service.NewQueryService(repo, ist)

		if _, err := svc.DailyActiveUsers(context.Background(), "2026-08-31"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// 2026-08-31 00:00 IST == 2026-08-30 18:30 UTC.
		wantStart := time.Date(2026, 8, 30, 18, 30, 0, 0, time.UTC)
		wantEnd := time.Date(2026, 8, 31, 18, 30, 0, 0, time.UTC)
		if !gotStart.Equal(wantStart) || !gotEnd.Equal(wantEnd) {
			t.Fatalf("expected bounds [%v, %v), got [%v, %v)", wantStart, wantEnd, gotStart, gotEnd)
		}
	})

	t.Run("rejects a malformed date", func(t *testing.T) {
		svc := service.NewQueryService(&fakeRepository{}, time.UTC)

		_, err := svc.DailyActiveUsers(context.Background(), "31-08-2026")
		if !errors.Is(err, service.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("rejects a future date", func(t *testing.T) {
		repo := &fakeRepository{
			countFunc: func(start, end time.Time) (int64, error) {
				t.Fatalf("repository should not be queried for a future date")
				return 0, nil
			},
		}
		svc := service.NewQueryService(repo, time.UTC)

		_, err := svc.DailyActiveUsers(context.Background(), "2150-01-01")
		if !errors.Is(err, service.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("does not reject today in a non-UTC service timezone", func(t *testing.T) {
		// Regression test: the future check must use the service's configured
		// timezone (via domain.DayBounds), not UTC. Between 18:30 and 23:59
		// UTC, "today" in Asia/Kolkata (UTC+5:30) is already "tomorrow" in
		// UTC, so a naive UTC-based comparison would wrongly reject it.
		ist, err := time.LoadLocation("Asia/Kolkata")
		if err != nil {
			t.Fatalf("failed to load Asia/Kolkata: %v", err)
		}
		todayIST := time.Now().In(ist).Format("2006-01-02")

		repo := &fakeRepository{countFunc: func(start, end time.Time) (int64, error) {
			return 5, nil
		}}
		svc := service.NewQueryService(repo, ist)

		count, err := svc.DailyActiveUsers(context.Background(), todayIST)
		if err != nil {
			t.Fatalf("expected today's IST date to be accepted, got error: %v", err)
		}
		if count != 5 {
			t.Fatalf("expected count 5, got %d", count)
		}
	})
}

func TestQueryService_MonthlyActiveUsers(t *testing.T) {
	t.Run("resolves UTC month bounds and returns the count", func(t *testing.T) {
		var gotStart, gotEnd time.Time
		repo := &fakeRepository{
			countFunc: func(start, end time.Time) (int64, error) {
				gotStart, gotEnd = start, end
				return 7, nil
			},
		}
		svc := service.NewQueryService(repo, time.UTC)

		count, err := svc.MonthlyActiveUsers(context.Background(), "2026-08")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 7 {
			t.Fatalf("expected count 7, got %d", count)
		}

		wantStart := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
		wantEnd := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
		if !gotStart.Equal(wantStart) || !gotEnd.Equal(wantEnd) {
			t.Fatalf("expected bounds [%v, %v), got [%v, %v)", wantStart, wantEnd, gotStart, gotEnd)
		}
	})

	t.Run("rejects a malformed month", func(t *testing.T) {
		svc := service.NewQueryService(&fakeRepository{}, time.UTC)

		_, err := svc.MonthlyActiveUsers(context.Background(), "2026/08")
		if !errors.Is(err, service.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("rejects a future month", func(t *testing.T) {
		repo := &fakeRepository{
			countFunc: func(start, end time.Time) (int64, error) {
				t.Fatalf("repository should not be queried for a future month")
				return 0, nil
			},
		}
		svc := service.NewQueryService(repo, time.UTC)

		_, err := svc.MonthlyActiveUsers(context.Background(), "2150-01")
		if !errors.Is(err, service.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})
}
