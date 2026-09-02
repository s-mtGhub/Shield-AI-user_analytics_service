package domain_test

import (
	"testing"
	"time"

	"user-analytics-service/internal/domain"
)

// ist returns the +05:30 India zone. Named-zone lookup needs the system
// tzdata, so fall back to a fixed zone (India observes no DST, so the two are
// equivalent for every instant asserted here) rather than skipping or flaking.
func ist(t *testing.T) *time.Location {
	t.Helper()
	if loc, err := time.LoadLocation("Asia/Kolkata"); err == nil {
		return loc
	}
	return time.FixedZone("IST", 5*3600+1800)
}

// assertUTCEqual checks an instant against a want value and also checks that
// the returned time.Time is actually carrying the UTC location.
func assertUTCEqual(t *testing.T, label string, got, want time.Time) {
	t.Helper()
	if !got.Equal(want) {
		t.Errorf("%s = %v, want %v", label, got, want)
	}
	if got.Location() != time.UTC {
		t.Errorf("%s location = %v, want UTC", label, got.Location())
	}
}

func TestDayBounds(t *testing.T) {
	kolkata := ist(t)

	tests := []struct {
		name      string
		date      string
		loc       *time.Location
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name:      "utc day",
			date:      "2026-08-31",
			loc:       time.UTC,
			wantStart: time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			// 2026-08-31 00:00 +05:30 == 2026-08-30 18:30 UTC. The start
			// instant lands on the *previous* calendar day in UTC, which is
			// what proves loc is honoured rather than ignored.
			name:      "positive offset zone shifts the window back 5h30m",
			date:      "2026-08-31",
			loc:       kolkata,
			wantStart: time.Date(2026, 8, 30, 18, 30, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 8, 31, 18, 30, 0, 0, time.UTC),
		},
		{
			name:      "month boundary",
			date:      "2026-08-01",
			loc:       time.UTC,
			wantStart: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "year boundary rolls into january",
			date:      "2026-12-31",
			loc:       time.UTC,
			wantStart: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "year boundary in a positive offset zone",
			date:      "2026-12-31",
			loc:       kolkata,
			wantStart: time.Date(2026, 12, 30, 18, 30, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 12, 31, 18, 30, 0, 0, time.UTC),
		},
		{
			name:      "leap day",
			date:      "2024-02-29",
			loc:       time.UTC,
			wantStart: time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "last day of a non leap february",
			date:      "2026-02-28",
			loc:       time.UTC,
			wantStart: time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start, end, err := domain.DayBounds(tc.date, tc.loc)
			if err != nil {
				t.Fatalf("DayBounds(%q) returned unexpected error: %v", tc.date, err)
			}
			assertUTCEqual(t, "start", start, tc.wantStart)
			assertUTCEqual(t, "end", end, tc.wantEnd)

			// A calendar day in a DST-free zone is exactly 24h, and the range
			// must be non-empty and half-open.
			if got := end.Sub(start); got != 24*time.Hour {
				t.Errorf("end - start = %v, want 24h", got)
			}
			if !start.Before(end) {
				t.Errorf("expected start %v before end %v", start, end)
			}
		})
	}
}

func TestDayBounds_Errors(t *testing.T) {
	tests := []struct {
		name string
		date string
	}{
		{name: "empty", date: ""},
		{name: "day first format", date: "31-08-2026"},
		{name: "slash separators", date: "2026/08/31"},
		{name: "month out of range", date: "2026-13-01"},
		{name: "day out of range for february", date: "2026-02-30"},
		{name: "february 29 in a non leap year", date: "2026-02-29"},
		{name: "zero month", date: "2026-00-01"},
		{name: "unpadded month", date: "2026-8-31"},
		{name: "full rfc3339 timestamp", date: "2026-08-31T00:00:00Z"},
		{name: "trailing whitespace", date: "2026-08-31 "},
		{name: "garbage", date: "not-a-date"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start, end, err := domain.DayBounds(tc.date, time.UTC)
			if err == nil {
				t.Fatalf("DayBounds(%q) = (%v, %v, nil), want an error", tc.date, start, end)
			}
			if !start.IsZero() {
				t.Errorf("start = %v, want the zero time on error", start)
			}
			if !end.IsZero() {
				t.Errorf("end = %v, want the zero time on error", end)
			}
		})
	}
}

func TestMonthBounds(t *testing.T) {
	kolkata := ist(t)

	tests := []struct {
		name      string
		month     string
		loc       *time.Location
		wantStart time.Time
		wantEnd   time.Time
		wantSpan  time.Duration
	}{
		{
			name:      "utc month",
			month:     "2026-08",
			loc:       time.UTC,
			wantStart: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
			wantSpan:  31 * 24 * time.Hour,
		},
		{
			name:      "december rolls the year",
			month:     "2026-12",
			loc:       time.UTC,
			wantStart: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
			wantSpan:  31 * 24 * time.Hour,
		},
		{
			name:      "january",
			month:     "2026-01",
			loc:       time.UTC,
			wantStart: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			wantSpan:  31 * 24 * time.Hour,
		},
		{
			// 2026-08-01 00:00 +05:30 == 2026-07-31 18:30 UTC: the month's
			// first instant sits in the previous calendar month in UTC.
			name:      "positive offset zone shifts the window back 5h30m",
			month:     "2026-08",
			loc:       kolkata,
			wantStart: time.Date(2026, 7, 31, 18, 30, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 8, 31, 18, 30, 0, 0, time.UTC),
			wantSpan:  31 * 24 * time.Hour,
		},
		{
			name:      "december in a positive offset zone",
			month:     "2026-12",
			loc:       kolkata,
			wantStart: time.Date(2026, 11, 30, 18, 30, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 12, 31, 18, 30, 0, 0, time.UTC),
			wantSpan:  31 * 24 * time.Hour,
		},
		{
			name:      "february in a leap year spans 29 days",
			month:     "2024-02",
			loc:       time.UTC,
			wantStart: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			wantSpan:  29 * 24 * time.Hour,
		},
		{
			name:      "february in a non leap year spans 28 days",
			month:     "2026-02",
			loc:       time.UTC,
			wantStart: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			wantSpan:  28 * 24 * time.Hour,
		},
		{
			name:      "thirty day month",
			month:     "2026-04",
			loc:       time.UTC,
			wantStart: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			wantSpan:  30 * 24 * time.Hour,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start, end, err := domain.MonthBounds(tc.month, tc.loc)
			if err != nil {
				t.Fatalf("MonthBounds(%q) returned unexpected error: %v", tc.month, err)
			}
			assertUTCEqual(t, "start", start, tc.wantStart)
			assertUTCEqual(t, "end", end, tc.wantEnd)

			if got := end.Sub(start); got != tc.wantSpan {
				t.Errorf("end - start = %v, want %v", got, tc.wantSpan)
			}
			if !start.Before(end) {
				t.Errorf("expected start %v before end %v", start, end)
			}
		})
	}
}

func TestMonthBounds_Errors(t *testing.T) {
	tests := []struct {
		name  string
		month string
	}{
		{name: "empty", month: ""},
		{name: "month out of range", month: "2026-13"},
		{name: "zero month", month: "2026-00"},
		// Layout element "01" requires a zero-padded two digit month, so the
		// unpadded form is rejected rather than being read as August.
		{name: "unpadded month", month: "2026-8"},
		{name: "too specific: includes a day", month: "2026-08-01"},
		{name: "two digit year", month: "26-08"},
		{name: "slash separator", month: "2026/08"},
		{name: "garbage", month: "garbage"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start, end, err := domain.MonthBounds(tc.month, time.UTC)
			if err == nil {
				t.Fatalf("MonthBounds(%q) = (%v, %v, nil), want an error", tc.month, start, end)
			}
			if !start.IsZero() {
				t.Errorf("start = %v, want the zero time on error", start)
			}
			if !end.IsZero() {
				t.Errorf("end = %v, want the zero time on error", end)
			}
		})
	}
}
