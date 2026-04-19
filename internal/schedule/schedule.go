// Package schedule computes the nightly window night-family runs in.
//
// The canonical schedule is a pair of "HH:MM" strings (window_start,
// window_end) interpreted in a configured IANA timezone. When end < start
// the window crosses midnight (the default 22:00 → 05:00 case).
//
// This package is pure: no globals, no I/O, no time.Now() baked in —
// callers pass a clock so tests can be deterministic.
package schedule

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// timeRE enforces the "HH:MM" shape used in config and the OpenAPI spec.
var timeRE = regexp.MustCompile(`^[0-2][0-9]:[0-5][0-9]$`)

// Schedule describes when the daemon runs.
type Schedule struct {
	// WindowStart and WindowEnd are 24-hour wall-clock strings in
	// TimeZone. WindowEnd < WindowStart means the window crosses
	// midnight (e.g. 22:00 → 05:00).
	WindowStart string `json:"window_start"`
	WindowEnd   string `json:"window_end"`

	// TimeZone is an IANA name. Empty means "system local".
	TimeZone string `json:"timezone"`

	// Cron, when non-empty, is a cron expression that overrides the
	// window. Not yet implemented — kept in the type so the API contract
	// is stable.
	Cron string `json:"cron,omitempty"`
}

// Default returns the canonical night-family schedule (22:00 → 05:00
// local). Matches docs/ROADMAP.md.
func Default() Schedule {
	return Schedule{
		WindowStart: "22:00",
		WindowEnd:   "05:00",
		TimeZone:    "",
	}
}

// Validate reports any rule violations. Callers should treat a non-nil
// error as a 400 from the API.
func (s Schedule) Validate() error {
	if err := validateHM("window_start", s.WindowStart); err != nil {
		return err
	}
	if err := validateHM("window_end", s.WindowEnd); err != nil {
		return err
	}
	if s.WindowStart == s.WindowEnd {
		return errors.New("window_start and window_end must differ")
	}
	if _, err := s.location(); err != nil {
		return err
	}
	return nil
}

func validateHM(field, v string) error {
	if !timeRE.MatchString(v) {
		return fmt.Errorf("%s %q: must match HH:MM", field, v)
	}
	h, m := parseHM(v)
	if h > 23 || m > 59 {
		return fmt.Errorf("%s %q: hour/minute out of range", field, v)
	}
	return nil
}

func (s Schedule) location() (*time.Location, error) {
	if s.TimeZone == "" {
		return time.Local, nil
	}
	return time.LoadLocation(s.TimeZone)
}

// parseHM converts an HH:MM string into (hour, minute).
func parseHM(s string) (int, int) {
	h, _ := strconv.Atoi(s[:2])
	m, _ := strconv.Atoi(s[3:])
	return h, m
}

// Clock is the abstraction Schedule uses so tests can pin time.
type Clock func() time.Time

// SystemClock returns a Clock backed by time.Now().
func SystemClock() Clock { return time.Now }

// Next returns the next (start, end) pair strictly at or after `now` in
// the schedule's timezone. If `now` is already inside a window, `start`
// is the start of the current window (and `end` its end).
func (s Schedule) Next(now time.Time) (start, end time.Time, err error) {
	if err := s.Validate(); err != nil {
		return time.Time{}, time.Time{}, err
	}
	loc, _ := s.location()
	localNow := now.In(loc)
	sh, sm := parseHM(s.WindowStart)
	eh, em := parseHM(s.WindowEnd)

	buildAt := func(day time.Time, h, m int) time.Time {
		return time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, loc)
	}

	// If the current window (possibly started yesterday) is still open,
	// use it.
	startToday := buildAt(localNow, sh, sm)
	endToday := buildAt(localNow, eh, em)

	if s.crossesMidnight() {
		// Today's window runs from startToday → (endToday + 24h).
		endOfTodays := endToday.Add(24 * time.Hour)
		// Yesterday's window runs from (startToday - 24h) → endToday.
		startYest := startToday.Add(-24 * time.Hour)

		if !localNow.Before(startYest) && localNow.Before(endToday) {
			return startYest, endToday, nil
		}
		if !localNow.Before(startToday) && localNow.Before(endOfTodays) {
			return startToday, endOfTodays, nil
		}
		// Not currently in a window: the next one starts at the next
		// startToday (today if still future, otherwise tomorrow).
		if localNow.Before(startToday) {
			return startToday, endOfTodays, nil
		}
		tomorrowStart := startToday.Add(24 * time.Hour)
		tomorrowEnd := tomorrowStart.Add(windowLen(sh, sm, eh, em))
		return tomorrowStart, tomorrowEnd, nil
	}

	// Non-crossing window.
	if !localNow.Before(startToday) && localNow.Before(endToday) {
		return startToday, endToday, nil
	}
	if localNow.Before(startToday) {
		return startToday, endToday, nil
	}
	tomorrowStart := startToday.Add(24 * time.Hour)
	tomorrowEnd := endToday.Add(24 * time.Hour)
	return tomorrowStart, tomorrowEnd, nil
}

// IsInWindow reports whether `now` falls inside the current or most
// recently-started window.
func (s Schedule) IsInWindow(now time.Time) bool {
	start, end, err := s.Next(now)
	if err != nil {
		return false
	}
	n := now.In(start.Location())
	return !n.Before(start) && n.Before(end)
}

func (s Schedule) crossesMidnight() bool {
	sh, sm := parseHM(s.WindowStart)
	eh, em := parseHM(s.WindowEnd)
	return (eh*60 + em) <= (sh*60 + sm)
}

// windowLen returns the wall-clock duration of a window that may cross
// midnight.
func windowLen(sh, sm, eh, em int) time.Duration {
	start := time.Duration(sh)*time.Hour + time.Duration(sm)*time.Minute
	end := time.Duration(eh)*time.Hour + time.Duration(em)*time.Minute
	if end <= start {
		end += 24 * time.Hour
	}
	return end - start
}

// Summary is the read-model the API returns at GET /schedule. It
// includes the computed next window for convenience.
type Summary struct {
	Schedule
	NextStart time.Time `json:"next_start"`
	NextEnd   time.Time `json:"next_end"`
	InWindow  bool      `json:"in_window"`
}

// Summarize computes the Summary at the given clock time.
func (s Schedule) Summarize(now time.Time) (Summary, error) {
	start, end, err := s.Next(now)
	if err != nil {
		return Summary{}, err
	}
	// Strip spaces/keep shape stable.
	s.WindowStart = strings.TrimSpace(s.WindowStart)
	s.WindowEnd = strings.TrimSpace(s.WindowEnd)
	return Summary{
		Schedule:  s,
		NextStart: start,
		NextEnd:   end,
		InWindow:  s.IsInWindow(now),
	}, nil
}
