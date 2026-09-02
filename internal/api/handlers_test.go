package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"user-analytics-service/internal/api"
	"user-analytics-service/internal/domain"
	"user-analytics-service/internal/service"
)

type fakeRepository struct {
	events []domain.LoginEvent
}

func (f *fakeRepository) SaveLoginEvent(_ context.Context, event domain.LoginEvent) error {
	f.events = append(f.events, event)
	return nil
}

func (f *fakeRepository) CountDistinctUsers(_ context.Context, start, end time.Time) (int64, error) {
	seen := map[string]struct{}{}
	for _, e := range f.events {
		if !e.Timestamp.Before(start) && e.Timestamp.Before(end) {
			seen[e.UserID] = struct{}{}
		}
	}
	return int64(len(seen)), nil
}

func newTestRouter(repo *fakeRepository) http.Handler {
	ingestionSvc := service.NewIngestionService(repo)
	querySvc := service.NewQueryService(repo, time.UTC)
	handler := api.NewHandler(ingestionSvc, querySvc)
	return api.NewRouter(handler)
}

func TestRecordLogin(t *testing.T) {
	repo := &fakeRepository{}
	router := newTestRouter(repo)

	body, _ := json.Marshal(api.LoginEventRequest{
		UserID:    "user-1",
		Timestamp: "2026-08-31T10:00:00Z",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(repo.events) != 1 {
		t.Fatalf("expected 1 saved event, got %d", len(repo.events))
	}
}

func TestRecordLogin_MissingUserID(t *testing.T) {
	router := newTestRouter(&fakeRepository{})

	body, _ := json.Marshal(api.LoginEventRequest{Timestamp: "2026-08-31T10:00:00Z"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDailyActiveUsers(t *testing.T) {
	repo := &fakeRepository{events: []domain.LoginEvent{
		{UserID: "user-1", Timestamp: time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)},
		{UserID: "user-2", Timestamp: time.Date(2026, 8, 31, 11, 0, 0, 0, time.UTC)},
		{UserID: "user-3", Timestamp: time.Date(2026, 9, 1, 1, 0, 0, 0, time.UTC)},
	}}
	router := newTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/daily-active-users?date=2026-08-31", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp api.DailyActiveUsersResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.UniqueActiveUsers != 2 {
		t.Fatalf("expected 2 unique active users, got %d", resp.UniqueActiveUsers)
	}
}

func TestDailyActiveUsers_MissingDate(t *testing.T) {
	router := newTestRouter(&fakeRepository{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/daily-active-users", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMonthlyActiveUsers(t *testing.T) {
	repo := &fakeRepository{events: []domain.LoginEvent{
		{UserID: "user-1", Timestamp: time.Date(2026, 8, 5, 9, 0, 0, 0, time.UTC)},
		{UserID: "user-2", Timestamp: time.Date(2026, 8, 20, 11, 0, 0, 0, time.UTC)},
		{UserID: "user-3", Timestamp: time.Date(2026, 9, 1, 1, 0, 0, 0, time.UTC)},
	}}
	router := newTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/monthly-active-users?month=2026-08", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp api.MonthlyActiveUsersResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.UniqueActiveUsers != 2 {
		t.Fatalf("expected 2 unique active users, got %d", resp.UniqueActiveUsers)
	}
}
