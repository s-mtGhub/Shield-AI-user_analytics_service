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
}
