package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"user-analytics-service/internal/domain"
	"user-analytics-service/internal/service"
)

type fakeRepository struct {
	events    []domain.LoginEvent
	saveErr   error
	countFunc func(start, end time.Time) (int64, error)
}

func (f *fakeRepository) SaveLoginEvent(_ context.Context, event domain.LoginEvent) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.events = append(f.events, event)
	return nil
}

func (f *fakeRepository) CountDistinctUsers(_ context.Context, start, end time.Time) (int64, error) {
	if f.countFunc != nil {
		return f.countFunc(start, end)
	}
	return 0, nil
}

func TestIngestionService_RecordLogin(t *testing.T) {
	ts := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)

	t.Run("persists a valid event", func(t *testing.T) {
		repo := &fakeRepository{}
		svc := service.NewIngestionService(repo)

		if err := svc.RecordLogin(context.Background(), "user-1", ts); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(repo.events) != 1 {
			t.Fatalf("expected 1 saved event, got %d", len(repo.events))
		}
		if repo.events[0].UserID != "user-1" || !repo.events[0].Timestamp.Equal(ts) {
			t.Fatalf("unexpected saved event: %+v", repo.events[0])
		}
	})

	t.Run("rejects empty user_id", func(t *testing.T) {
		repo := &fakeRepository{}
		svc := service.NewIngestionService(repo)

		err := svc.RecordLogin(context.Background(), "  ", ts)
		if !errors.Is(err, service.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
		if len(repo.events) != 0 {
			t.Fatalf("expected no saved events, got %d", len(repo.events))
		}
	})

	t.Run("rejects zero timestamp", func(t *testing.T) {
		repo := &fakeRepository{}
		svc := service.NewIngestionService(repo)

		err := svc.RecordLogin(context.Background(), "user-1", time.Time{})
		if !errors.Is(err, service.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("propagates repository errors", func(t *testing.T) {
		wantErr := errors.New("db unavailable")
		repo := &fakeRepository{saveErr: wantErr}
		svc := service.NewIngestionService(repo)

		err := svc.RecordLogin(context.Background(), "user-1", ts)
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})
}
