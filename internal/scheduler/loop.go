// Package scheduler is the background loop that fires a night when
// the configured window opens.
//
// v1 is a tick loop: every tick, check whether we're inside the
// window and whether we've already run a night for this window. If
// both answers line up, trigger. That's it — no cron, no overlap, no
// catching up on missed nights.
//
// The heavy lifting (plan, dispatch, record) lives in runner.TriggerNight.
// This package only owns the "when to fire" decision.
package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bupd/night-family/internal/runner"
	"github.com/bupd/night-family/internal/schedule"
)

// Options configure a Loop.
type Options struct {
	Schedule *schedule.Schedule
	Runner   *runner.Runner
	Logger   *slog.Logger
	// Tick is the poll interval. 1m is plenty for the nightly cadence;
	// tests override to nanoseconds.
	Tick time.Duration
	// Clock, when set, replaces time.Now for testing.
	Clock func() time.Time
	// TriggerOpts is passed through to runner.TriggerNight on every
	// auto-fire. Useful for e.g. setting a default budget.
	TriggerOpts runner.NightOptions
}

// Loop drives a ticker; it fires TriggerNight at most once per window
// start. Call Start to spawn the goroutine; it stops when ctx is
// cancelled.
type Loop struct {
	opts     Options
	mu       sync.Mutex
	lastFire time.Time
	once     sync.Once
}

// New returns a Loop ready to be Start()-ed.
func New(opts Options) *Loop {
	if opts.Tick == 0 {
		opts.Tick = time.Minute
	}
	if opts.Clock == nil {
		opts.Clock = time.Now
	}
	return &Loop{opts: opts}
}

// Start spawns the loop. Safe to call twice — second call is a no-op.
func (l *Loop) Start(ctx context.Context) {
	l.once.Do(func() {
		go l.run(ctx)
	})
}

// LastFire returns the wall time of the most recent auto-fire, or
// zero if nothing has fired yet.
func (l *Loop) LastFire() time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.lastFire
}

func (l *Loop) run(ctx context.Context) {
	t := time.NewTicker(l.opts.Tick)
	defer t.Stop()
	l.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			l.opts.Logger.Info("scheduler stopped")
			return
		case <-t.C:
			l.tick(ctx)
		}
	}
}

func (l *Loop) tick(ctx context.Context) {
	now := l.opts.Clock()
	start, _, err := l.opts.Schedule.Next(now)
	if err != nil {
		l.opts.Logger.Warn("scheduler: schedule invalid", "err", err)
		return
	}
	if !l.opts.Schedule.IsInWindow(now) {
		return
	}
	l.mu.Lock()
	already := !l.lastFire.Before(start)
	l.mu.Unlock()
	if already {
		return
	}
	l.opts.Logger.Info("scheduler: firing night", "window_start", start)
	res, err := l.opts.Runner.TriggerNight(ctx, l.opts.Schedule, l.opts.TriggerOpts)
	if err != nil {
		l.opts.Logger.Error("scheduler: TriggerNight failed", "err", err)
		return
	}
	l.mu.Lock()
	l.lastFire = now
	l.mu.Unlock()
	l.opts.Logger.Info("scheduler: night fired", "night", res.ID, "runs", len(res.Runs))
}
