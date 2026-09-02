package api

const contentTypeJSON = "application/json"

const (
	pathLoginEvent         = "/api/v1/events/login"
	pathDailyActiveUsers   = "/api/v1/analytics/daily-active-users"
	pathMonthlyActiveUsers = "/api/v1/analytics/monthly-active-users"
)

const (
	queryParamDate  = "date"
	queryParamMonth = "month"
)

const (
	msgInvalidJSONBody        = "invalid JSON body"
	msgInvalidTimestamp       = "timestamp must be RFC3339, e.g. 2026-08-31T10:15:00Z"
	msgMissingDateParam       = "date query parameter is required (YYYY-MM-DD)"
	msgMissingMonthParam      = "month query parameter is required (YYYY-MM)"
	msgRecordLoginFailed      = "failed to record login event"
	msgDailyActiveUsersFail   = "failed to compute daily active users"
	msgMonthlyActiveUsersFail = "failed to compute monthly active users"
	msgFutureTimestamp        = "timestamp cannot be in the future"
)
