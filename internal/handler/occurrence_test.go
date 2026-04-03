package handler

import (
	"testing"
	"time"
)

// ---------- generateRecurrenceDates ----------

func TestGenerateRecurrenceDates_Daily(t *testing.T) {
	start := time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC)
	until := time.Date(2026, 4, 5, 23, 59, 59, 0, time.UTC)

	dates := generateRecurrenceDates(start, until, "daily")

	if len(dates) != 5 {
		t.Fatalf("daily: expected 5 dates, got %d", len(dates))
	}
	for i, d := range dates {
		expected := start.AddDate(0, 0, i)
		if !d.Equal(expected) {
			t.Errorf("daily[%d]: want %v, got %v", i, expected, d)
		}
	}
}

func TestGenerateRecurrenceDates_Weekly(t *testing.T) {
	start := time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC)
	// 3 weeks later
	until := time.Date(2026, 4, 22, 23, 59, 59, 0, time.UTC)

	dates := generateRecurrenceDates(start, until, "weekly")

	// start, +7d, +14d, +21d = 4 dates
	if len(dates) != 4 {
		t.Fatalf("weekly: expected 4 dates, got %d", len(dates))
	}
	for i, d := range dates {
		expected := start.AddDate(0, 0, 7*i)
		if !d.Equal(expected) {
			t.Errorf("weekly[%d]: want %v, got %v", i, expected, d)
		}
	}
}

func TestGenerateRecurrenceDates_Biweekly(t *testing.T) {
	start := time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC)
	until := time.Date(2026, 5, 15, 23, 59, 59, 0, time.UTC)

	dates := generateRecurrenceDates(start, until, "biweekly")

	// start(Apr 1), +14d(Apr 15), +28d(Apr 29), +42d(May 13) = 4 dates
	// +56d = May 27 which is after until (May 15)
	if len(dates) != 4 {
		t.Fatalf("biweekly: expected 4 dates, got %d", len(dates))
	}
	for i, d := range dates {
		expected := start.AddDate(0, 0, 14*i)
		if !d.Equal(expected) {
			t.Errorf("biweekly[%d]: want %v, got %v", i, expected, d)
		}
	}
}

func TestGenerateRecurrenceDates_Monthly(t *testing.T) {
	start := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	until := time.Date(2026, 5, 15, 23, 59, 59, 0, time.UTC)

	dates := generateRecurrenceDates(start, until, "monthly")

	// Jan 15, Feb 15, Mar 15, Apr 15, May 15 = 5 dates
	if len(dates) != 5 {
		t.Fatalf("monthly: expected 5 dates, got %d", len(dates))
	}
	for i, d := range dates {
		expected := start.AddDate(0, i, 0)
		if !d.Equal(expected) {
			t.Errorf("monthly[%d]: want %v, got %v", i, expected, d)
		}
	}
}

func TestGenerateRecurrenceDates_UntilEqualsStart(t *testing.T) {
	start := time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC)
	// until is before the next occurrence but on the same day
	until := time.Date(2026, 4, 1, 23, 59, 59, 0, time.UTC)

	dates := generateRecurrenceDates(start, until, "daily")

	// Only start should be returned because the next day (Apr 2) > until (Apr 1 23:59)
	if len(dates) != 1 {
		t.Fatalf("until==start day: expected 1 date, got %d", len(dates))
	}
	if !dates[0].Equal(start) {
		t.Errorf("expected %v, got %v", start, dates[0])
	}
}

func TestGenerateRecurrenceDates_UntilBeforeStart(t *testing.T) {
	start := time.Date(2026, 4, 10, 8, 0, 0, 0, time.UTC)
	until := time.Date(2026, 4, 5, 23, 59, 59, 0, time.UTC) // before start

	dates := generateRecurrenceDates(start, until, "daily")

	// The function always includes start, then any subsequent date that is > until
	// won't be added. So only start is returned.
	if len(dates) != 1 {
		t.Fatalf("until before start: expected 1 date, got %d", len(dates))
	}
	if !dates[0].Equal(start) {
		t.Errorf("expected %v, got %v", start, dates[0])
	}
}

func TestGenerateRecurrenceDates_UnknownRepeatMode(t *testing.T) {
	start := time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC)
	until := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)

	for _, mode := range []string{"", "yearly", "bogus"} {
		dates := generateRecurrenceDates(start, until, mode)
		if len(dates) != 1 {
			t.Errorf("mode %q: expected 1 date (just start), got %d", mode, len(dates))
		}
		if len(dates) > 0 && !dates[0].Equal(start) {
			t.Errorf("mode %q: expected start date, got %v", mode, dates[0])
		}
	}
}

func TestGenerateRecurrenceDates_CapDoesNotExceedUntil(t *testing.T) {
	start := time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC)
	until := time.Date(2026, 4, 10, 8, 0, 0, 0, time.UTC)

	dates := generateRecurrenceDates(start, until, "weekly")

	// Apr 1, Apr 8 = 2 dates. Apr 15 would exceed until.
	if len(dates) != 2 {
		t.Fatalf("cap: expected 2 dates, got %d", len(dates))
	}
	for _, d := range dates {
		if d.After(until) {
			t.Errorf("date %v exceeds until %v", d, until)
		}
	}
}

func TestGenerateRecurrenceDates_DailyLargeRange(t *testing.T) {
	start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	until := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)

	dates := generateRecurrenceDates(start, until, "daily")

	// 365 days in 2026 (non-leap year)
	if len(dates) != 365 {
		t.Fatalf("large daily: expected 365 dates, got %d", len(dates))
	}

	// Verify no date exceeds until
	for i, d := range dates {
		if d.After(until) {
			t.Errorf("date[%d] %v exceeds until %v", i, d, until)
		}
	}
}

func TestGenerateRecurrenceDates_FirstDateIsAlwaysStart(t *testing.T) {
	start := time.Date(2026, 6, 15, 14, 30, 0, 0, time.UTC)
	until := time.Date(2026, 9, 15, 23, 59, 59, 0, time.UTC)

	for _, mode := range []string{"daily", "weekly", "biweekly", "monthly"} {
		dates := generateRecurrenceDates(start, until, mode)
		if len(dates) == 0 {
			t.Fatalf("mode %q: got 0 dates", mode)
		}
		if !dates[0].Equal(start) {
			t.Errorf("mode %q: first date should be start %v, got %v", mode, start, dates[0])
		}
	}
}

// ---------- newRecurrenceID ----------

func TestNewRecurrenceID_NonEmpty(t *testing.T) {
	id := newRecurrenceID()
	if id == "" {
		t.Fatal("newRecurrenceID returned empty string")
	}
}

func TestNewRecurrenceID_UniqueValues(t *testing.T) {
	id1 := newRecurrenceID()
	id2 := newRecurrenceID()
	if id1 == id2 {
		t.Fatalf("two calls to newRecurrenceID returned the same value: %q", id1)
	}
}

func TestNewRecurrenceID_HexFormat(t *testing.T) {
	id := newRecurrenceID()
	// 16 bytes => 32 hex characters
	if len(id) != 32 {
		t.Errorf("expected 32-character hex string, got %d characters: %q", len(id), id)
	}
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("unexpected character %q in recurrence ID %q", string(c), id)
			break
		}
	}
}
