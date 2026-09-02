package domain

import "time"

// LoginEvent represents a single user login activity record.
type LoginEvent struct {
	UserID    string
	Timestamp time.Time
}
