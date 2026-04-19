package schedule

import (
	"testing"
	"time"
)

func mustLoc(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("LoadLocation(%q): %v", name, err)
	}
	return loc
}

func TestDefaultSchedule(t *testing.T) {
	s := Default()
	if err := s.Validate(); err != nil {
		t.Fatalf("default invalid: %v", err)
	}
	if !s.crossesMidnight() {
		t.Errorf("default window should cross midnight")
	}
}

func TestValidate(t *testing.T) {
	cases := map[string]Schedule{
		"bad start":    {WindowStart: "25:00", WindowEnd: "05:00"},
		"bad end":      {WindowStart: "22:00", WindowEnd: "nope"},
		"equal":        {WindowStart: "22:00", WindowEnd: "22:00"},
		"bad timezone": {WindowStart: "22:00", WindowEnd: "05:00", TimeZone: "Mars/Olympus"},
	}
	for name, s := range cases {
		if err := s.Validate(); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	ok := Schedule{WindowStart: "22:00", WindowEnd: "05:00", TimeZone: "Europe/Berlin"}
	if err := ok.Validate(); err != nil {
		t.Errorf("good schedule failed: %v", err)
	}
}

func TestNextBeforeWindow(t *testing.T) {
	loc := mustLoc(t, "Europe/Berlin")
	s := Schedule{WindowStart: "22:00", WindowEnd: "05:00", TimeZone: "Europe/Berlin"}
	now := time.Date(2026, 4, 17, 20, 0, 0, 0, loc) // 20:00 before window
	start, end, err := s.Next(now)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if start.Hour() != 22 || start.Minute() != 0 {
		t.Errorf("start = %v, want 22:00", start)
	}
	wantEnd := time.Date(2026, 4, 18, 5, 0, 0, 0, loc)
	if !end.Equal(wantEnd) {
		t.Errorf("end = %v, want %v", end, wantEnd)
	}
	if s.IsInWindow(now) {
		t.Errorf("IsInWindow true for pre-window time")
	}
}

func TestNextDuringWindow(t *testing.T) {
	loc := mustLoc(t, "Europe/Berlin")
	s := Schedule{WindowStart: "22:00", WindowEnd: "05:00", TimeZone: "Europe/Berlin"}

	// Inside window, past-midnight side
	now := time.Date(2026, 4, 18, 2, 30, 0, 0, loc)
	if !s.IsInWindow(now) {
		t.Errorf("2:30 should be in the 22:00→05:00 window")
	}
	start, end, err := s.Next(now)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	// Start should be 22:00 of the previous day.
	want := time.Date(2026, 4, 17, 22, 0, 0, 0, loc)
	if !start.Equal(want) {
		t.Errorf("start = %v, want %v", start, want)
	}
	if end.Hour() != 5 {
		t.Errorf("end = %v, want 05:00", end)
	}
}

func TestNextAfterWindow(t *testing.T) {
	loc := mustLoc(t, "Europe/Berlin")
	s := Schedule{WindowStart: "22:00", WindowEnd: "05:00", TimeZone: "Europe/Berlin"}
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, loc) // noon
	start, _, err := s.Next(now)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if start.Day() != 18 || start.Hour() != 22 {
		t.Errorf("start = %v, want 2026-04-18 22:00", start)
	}
	if s.IsInWindow(now) {
		t.Errorf("noon should not be in window")
	}
}

func TestNonCrossingWindow(t *testing.T) {
	loc := mustLoc(t, "Europe/Berlin")
	s := Schedule{WindowStart: "09:00", WindowEnd: "17:00", TimeZone: "Europe/Berlin"}
	now := time.Date(2026, 4, 17, 13, 0, 0, 0, loc)
	if !s.IsInWindow(now) {
		t.Errorf("13:00 should be in 09:00→17:00 window")
	}
	before := time.Date(2026, 4, 17, 8, 0, 0, 0, loc)
	start, end, err := s.Next(before)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if start.Hour() != 9 || end.Hour() != 17 {
		t.Errorf("start/end = %v/%v", start, end)
	}
}

func TestSummarize(t *testing.T) {
	loc := mustLoc(t, "Europe/Berlin")
	s := Default()
	s.TimeZone = "Europe/Berlin"
	sum, err := s.Summarize(time.Date(2026, 4, 17, 23, 0, 0, 0, loc))
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if !sum.InWindow {
		t.Errorf("expected InWindow=true")
	}
	if sum.NextStart.IsZero() || sum.NextEnd.IsZero() {
		t.Errorf("times not filled")
	}
}
