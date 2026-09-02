package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"user-analytics-service/internal/api"
	"user-analytics-service/internal/domain"
	"user-analytics-service/internal/repository"
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

// erroringRepository fails every call with err. Because IngestionService and
// QueryService are concrete structs over repository.Repository, a repository
// returning a non-ErrInvalidInput error is the only way to reach the handlers'
// http.StatusInternalServerError branches.
type erroringRepository struct {
	err error
}

func (r *erroringRepository) SaveLoginEvent(_ context.Context, _ domain.LoginEvent) error {
	return r.err
}

func (r *erroringRepository) CountDistinctUsers(_ context.Context, _, _ time.Time) (int64, error) {
	return 0, r.err
}

func newTestRouter(repo *fakeRepository) http.Handler {
	return newRouterFor(repo)
}

func newRouterFor(repo repository.Repository) http.Handler {
	ingestionSvc := service.NewIngestionService(repo)
	querySvc := service.NewQueryService(repo, time.UTC)
	handler := api.NewHandler(ingestionSvc, querySvc)
	return api.NewRouter(handler)
}

// doRequest runs one request against router and returns the recorder.
func doRequest(router http.Handler, method, target string, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, target, nil)
	} else {
		req = httptest.NewRequest(method, target, bytes.NewReader([]byte(body)))
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// assertJSONError asserts a JSON ErrorResponse body with the given status. When
// wantMessage is non-empty it must match exactly; otherwise wantPrefix applies.
func assertJSONError(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantMessage, wantPrefix string) {
	t.Helper()

	if rec.Code != wantStatus {
		t.Fatalf("expected status %d, got %d: %s", wantStatus, rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", got)
	}

	var resp api.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode error response %q: %v", rec.Body.String(), err)
	}
	if wantMessage != "" && resp.Error != wantMessage {
		t.Fatalf("expected error %q, got %q", wantMessage, resp.Error)
	}
	if wantPrefix != "" && !strings.HasPrefix(resp.Error, wantPrefix) {
		t.Fatalf("expected error with prefix %q, got %q", wantPrefix, resp.Error)
	}
	if resp.Error == "" {
		t.Fatalf("expected a non-empty error message, got %q", rec.Body.String())
	}
}

// invalidInputPrefix is the prefix every service.ErrInvalidInput message carries.
var invalidInputPrefix = service.ErrInvalidInput.Error() + ": "

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

func TestRecordLogin_BadRequests(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantMessage string
		wantPrefix  string
	}{
		{
			name:        "malformed json",
			body:        `{"user_id": "user-1"`,
			wantMessage: "invalid JSON body",
		},
		{
			name:        "not json at all",
			body:        "not json",
			wantMessage: "invalid JSON body",
		},
		{
			name:        "empty body",
			body:        "",
			wantMessage: "invalid JSON body",
		},
		{
			name:        "valid json wrong shape",
			body:        `{"user_id": 123}`,
			wantMessage: "invalid JSON body",
		},
		{
			name:        "json array instead of object",
			body:        `["user-1"]`,
			wantMessage: "invalid JSON body",
		},
		{
			name:        "timestamp not rfc3339",
			body:        `{"user_id": "user-1", "timestamp": "31-08-2026"}`,
			wantMessage: "timestamp must be RFC3339, e.g. 2026-08-31T10:15:00Z",
		},
		{
			name:        "timestamp date only",
			body:        `{"user_id": "user-1", "timestamp": "2026-08-31"}`,
			wantMessage: "timestamp must be RFC3339, e.g. 2026-08-31T10:15:00Z",
		},
		{
			name:        "future timestamp is invalid input",
			body:        `{"user_id": "user-1", "timestamp": "2150-08-31T10:00:00Z"}`,
			wantMessage: "timestamp cannot be in the future",
		},
		{
			name:        "empty user id is invalid input",
			body:        `{"user_id": "", "timestamp": "2026-08-31T10:00:00Z"}`,
			wantMessage: invalidInputPrefix + "user_id is required",
		},
		{
			name:        "whitespace user id is invalid input",
			body:        `{"user_id": "   ", "timestamp": "2026-08-31T10:00:00Z"}`,
			wantMessage: invalidInputPrefix + "user_id is required",
		},
		{
			name:        "user id absent is invalid input",
			body:        `{"timestamp": "2026-08-31T10:00:00Z"}`,
			wantMessage: invalidInputPrefix + "user_id is required",
		},
		{
			name:        "zero timestamp is invalid input",
			body:        `{"user_id": "user-1", "timestamp": "0001-01-01T00:00:00Z"}`,
			wantMessage: invalidInputPrefix + "timestamp is required",
			wantPrefix:  invalidInputPrefix,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{}
			rec := doRequest(newTestRouter(repo), http.MethodPost, "/api/v1/events/login", tc.body)

			assertJSONError(t, rec, http.StatusBadRequest, tc.wantMessage, tc.wantPrefix)
			if len(repo.events) != 0 {
				t.Fatalf("expected no saved events, got %d", len(repo.events))
			}
		})
	}
}

func TestRecordLogin_RepositoryFailure(t *testing.T) {
	router := newRouterFor(&erroringRepository{err: errors.New("boom")})

	rec := doRequest(router, http.MethodPost, "/api/v1/events/login",
		`{"user_id": "user-1", "timestamp": "2026-08-31T10:00:00Z"}`)

	assertJSONError(t, rec, http.StatusInternalServerError, "failed to record login event", "")
}

func TestRecordLogin_DefaultsTimestampToNow(t *testing.T) {
	repo := &fakeRepository{}
	before := time.Now().UTC().Add(-2 * time.Second)

	rec := doRequest(newTestRouter(repo), http.MethodPost, "/api/v1/events/login", `{"user_id": "user-1"}`)

	after := time.Now().UTC().Add(2 * time.Second)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", got)
	}

	var resp api.LoginEventResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.UserID != "user-1" {
		t.Fatalf("expected user_id user-1, got %q", resp.UserID)
	}

	ts, err := time.Parse(time.RFC3339, resp.Timestamp)
	if err != nil {
		t.Fatalf("expected an RFC3339 timestamp, got %q: %v", resp.Timestamp, err)
	}
	if ts.Before(before) || ts.After(after) {
		t.Fatalf("expected server-generated timestamp within [%s, %s], got %s",
			before.Format(time.RFC3339), after.Format(time.RFC3339), ts.Format(time.RFC3339))
	}

	if len(repo.events) != 1 {
		t.Fatalf("expected 1 saved event, got %d", len(repo.events))
	}
	if saved := repo.events[0].Timestamp; saved.Before(before) || saved.After(after) {
		t.Fatalf("expected saved timestamp within [%s, %s], got %s",
			before.Format(time.RFC3339), after.Format(time.RFC3339), saved.Format(time.RFC3339))
	}
}

// TestRecordLogin_EchoesUntrimmedUserID pins current behaviour: the handler
// echoes the raw request user_id while the service persists the trimmed value,
// so the 201 response can disagree with what was stored. See the report note.
func TestRecordLogin_EchoesUntrimmedUserID(t *testing.T) {
	repo := &fakeRepository{}

	rec := doRequest(newTestRouter(repo), http.MethodPost, "/api/v1/events/login",
		`{"user_id": "  user-1  ", "timestamp": "2026-08-31T10:00:00Z"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp api.LoginEventResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.UserID != "  user-1  " {
		t.Fatalf("expected the echoed user_id %q, got %q", "  user-1  ", resp.UserID)
	}
	if len(repo.events) != 1 {
		t.Fatalf("expected 1 saved event, got %d", len(repo.events))
	}
	if repo.events[0].UserID != "user-1" {
		t.Fatalf("expected the stored user_id to be trimmed to %q, got %q", "user-1", repo.events[0].UserID)
	}
}

func TestAnalyticsHandlers_BadRequests(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		wantMessage string
		wantPrefix  string
	}{
		{
			name:        "daily missing date param",
			target:      "/api/v1/analytics/daily-active-users",
			wantMessage: "date query parameter is required (YYYY-MM-DD)",
		},
		{
			name:        "daily empty date param",
			target:      "/api/v1/analytics/daily-active-users?date=",
			wantMessage: "date query parameter is required (YYYY-MM-DD)",
		},
		{
			name:       "daily unparseable date",
			target:     "/api/v1/analytics/daily-active-users?date=31-08-2026",
			wantPrefix: invalidInputPrefix,
		},
		{
			name:       "daily month granularity date",
			target:     "/api/v1/analytics/daily-active-users?date=2026-08",
			wantPrefix: invalidInputPrefix,
		},
		{
			name:       "daily out of range date",
			target:     "/api/v1/analytics/daily-active-users?date=2026-13-45",
			wantPrefix: invalidInputPrefix,
		},
		{
			name:        "monthly missing month param",
			target:      "/api/v1/analytics/monthly-active-users",
			wantMessage: "month query parameter is required (YYYY-MM)",
		},
		{
			name:        "monthly empty month param",
			target:      "/api/v1/analytics/monthly-active-users?month=",
			wantMessage: "month query parameter is required (YYYY-MM)",
		},
		{
			name:       "monthly unparseable month",
			target:     "/api/v1/analytics/monthly-active-users?month=August-2026",
			wantPrefix: invalidInputPrefix,
		},
		{
			name:       "monthly day granularity month",
			target:     "/api/v1/analytics/monthly-active-users?month=2026-08-31",
			wantPrefix: invalidInputPrefix,
		},
		{
			name:       "monthly out of range month",
			target:     "/api/v1/analytics/monthly-active-users?month=2026-13",
			wantPrefix: invalidInputPrefix,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := doRequest(newTestRouter(&fakeRepository{}), http.MethodGet, tc.target, "")
			assertJSONError(t, rec, http.StatusBadRequest, tc.wantMessage, tc.wantPrefix)
		})
	}
}

func TestAnalyticsHandlers_RepositoryFailure(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		wantMessage string
	}{
		{
			name:        "daily active users",
			target:      "/api/v1/analytics/daily-active-users?date=2026-08-31",
			wantMessage: "failed to compute daily active users",
		},
		{
			name:        "monthly active users",
			target:      "/api/v1/analytics/monthly-active-users?month=2026-08",
			wantMessage: "failed to compute monthly active users",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := newRouterFor(&erroringRepository{err: errors.New("boom")})
			rec := doRequest(router, http.MethodGet, tc.target, "")
			assertJSONError(t, rec, http.StatusInternalServerError, tc.wantMessage, "")
		})
	}
}

func TestRouter_MethodAndPathRouting(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		target     string
		wantStatus int
	}{
		{
			name:       "get on login path is method not allowed",
			method:     http.MethodGet,
			target:     "/api/v1/events/login",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "delete on login path is method not allowed",
			method:     http.MethodDelete,
			target:     "/api/v1/events/login",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "post on daily active users is method not allowed",
			method:     http.MethodPost,
			target:     "/api/v1/analytics/daily-active-users?date=2026-08-31",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "post on monthly active users is method not allowed",
			method:     http.MethodPost,
			target:     "/api/v1/analytics/monthly-active-users?month=2026-08",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "unknown path is not found",
			method:     http.MethodGet,
			target:     "/api/v1/unknown",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "root path is not found",
			method:     http.MethodGet,
			target:     "/",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "login sub path is not found",
			method:     http.MethodPost,
			target:     "/api/v1/events/login/extra",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{}
			rec := doRequest(newTestRouter(repo), tc.method, tc.target, "")

			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", tc.wantStatus, rec.Code, rec.Body.String())
			}
			if len(repo.events) != 0 {
				t.Fatalf("expected no saved events, got %d", len(repo.events))
			}
		})
	}
}
