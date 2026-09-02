package domain

import (
	"fmt"
	"time"
)

// DayBounds returns the [start, end) UTC instant range covering the calendar
// day identified by date (format "2006-01-02"), interpreted in loc.
func DayBounds(date string, loc *time.Location) (time.Time, time.Time, error) {
	d, err := time.ParseInLocation(dateLayout, date, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid date %q, expected YYYY-MM-DD: %w", date, err)
	}
	start := d.UTC()
	end := d.AddDate(0, 0, 1).UTC()
	return start, end, nil
}

// MonthBounds returns the [start, end) UTC instant range covering the
// calendar month identified by month (format "2006-01"), interpreted in loc.
func MonthBounds(month string, loc *time.Location) (time.Time, time.Time, error) {
	m, err := time.ParseInLocation(monthLayout, month, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid month %q, expected YYYY-MM: %w", month, err)
	}
	start := m.UTC()
	end := m.AddDate(0, 1, 0).UTC()
	return start, end, nil
}
