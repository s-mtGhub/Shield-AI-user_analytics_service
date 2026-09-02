package service

import (
	"context"
	"fmt"
	"time"

	"user-analytics-service/internal/domain"
	"user-analytics-service/internal/repository"
)

// QueryService answers activity aggregate questions.
type QueryService struct {
	repo repository.Repository
	loc  *time.Location
}

// NewQueryService builds a QueryService backed by repo, resolving calendar
// day/month boundaries in loc before converting them to UTC instants.
func NewQueryService(repo repository.Repository, loc *time.Location) *QueryService {
	return &QueryService{repo: repo, loc: loc}
}

// DailyActiveUsers returns the count of distinct users active on date
// (format "2006-01-02"), interpreted in the service timezone.
func (s *QueryService) DailyActiveUsers(ctx context.Context, date string) (int64, error) {
	start, end, err := domain.DayBounds(date, s.loc)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	// if start.After(time.Now().UTC()) {
	// 	return 0, fmt.Errorf("%w: %s", ErrInvalidInput, msgDateCannotBeFuture)
	// }
	return s.repo.CountDistinctUsers(ctx, start, end)
}

// MonthlyActiveUsers returns the count of distinct users active during month
// (format "2006-01"), interpreted in the service timezone.
func (s *QueryService) MonthlyActiveUsers(ctx context.Context, month string) (int64, error) {
	start, end, err := domain.MonthBounds(month, s.loc)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	// if start.After(time.Now().UTC()) {
	// 	return 0, fmt.Errorf("%w: %s", ErrInvalidInput, msgMonthCannotBeFuture)
	// }
	return s.repo.CountDistinctUsers(ctx, start, end)
}
