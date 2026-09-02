package api

// LoginEventRequest is the payload for POST /api/v1/events/login.
type LoginEventRequest struct {
	UserID string `json:"user_id"`
	// Timestamp is RFC3339 (e.g. "2026-08-31T10:15:00Z"). Optional; defaults
	// to the server's current time when omitted.
	Timestamp string `json:"timestamp,omitempty"`
}

// LoginEventResponse acknowledges a recorded login event.
type LoginEventResponse struct {
	UserID    string `json:"user_id"`
	Timestamp string `json:"timestamp"`
}

// DailyActiveUsersResponse is the payload for GET /api/v1/analytics/daily-active-users.
type DailyActiveUsersResponse struct {
	Date              string `json:"date"`
	UniqueActiveUsers int64  `json:"unique_active_users"`
}

// MonthlyActiveUsersResponse is the payload for GET /api/v1/analytics/monthly-active-users.
type MonthlyActiveUsersResponse struct {
	Month             string `json:"month"`
	UniqueActiveUsers int64  `json:"unique_active_users"`
}

// ErrorResponse is the standard error payload for non-2xx responses.
type ErrorResponse struct {
	Error string `json:"error"`
}
