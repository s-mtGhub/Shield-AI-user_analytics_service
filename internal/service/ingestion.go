package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"user-analytics-service/internal/domain"
	"user-analytics-service/internal/repository"
)

// ErrInvalidInput wraps input validation failures so callers (e.g. HTTP
// handlers) can distinguish a 400 from a downstream/500 error.
var ErrInvalidInput = errors.New("invalid input")

// IngestionService records user activity events.
type IngestionService struct {
	repo repository.Repository
}

// NewIngestionService builds an IngestionService backed by repo.
func NewIngestionService(repo repository.Repository) *IngestionService {
	return &IngestionService{repo: repo}
}

// RecordLogin validates and persists a login event for userID at timestamp.
func (s *IngestionService) RecordLogin(ctx context.Context, userID string, timestamp time.Time) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return fmt.Errorf("%w: %s", ErrInvalidInput, msgUserIDRequired)
	}
	if timestamp.IsZero() {
		return fmt.Errorf("%w: %s", ErrInvalidInput, msgTimestampRequired)
	}

	event := domain.LoginEvent{
		UserID:    userID,
		Timestamp: timestamp.UTC(),
	}
	return s.repo.SaveLoginEvent(ctx, event)
}
