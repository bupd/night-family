package scheduler

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bupd/night-family/internal/duty"
	"github.com/bupd/night-family/internal/family"
	"github.com/bupd/night-family/internal/provider"
	"github.com/bupd/night-family/internal/runner"
	"github.com/bupd/night-family/internal/schedule"
	"github.com/bupd/night-family/internal/storage"
)

func TestTickInsideWindowFires(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Berlin")
	sched := schedule.Schedule{WindowStart: "22:00", WindowEnd: "05:00", TimeZone: "Europe/Berlin"}
	// pinned clock: 23:00 (inside window)
	now := time.Date(2026, 4, 17, 23, 0, 0, 0, loc)
	var fires int32
	wrapped := wrapCountRunner(t, &fires)

	l := New(Options{
		Schedule: &sched,
		Runner:   wrapped,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Tick:     time.Nanosecond,
		Clock:    func() time.Time { return now },
	})
	// Call tick twice — second should be a no-op because we've already fired for this window.
	l.tick(context.Background())
	l.tick(context.Background())
	if got := atomic.LoadInt32(&fires); got != 1 {
		t.Fatalf("fires = %d, want 1", got)
	}
}

func TestTickOutsideWindowDoesNotFire(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Berlin")
	sched := schedule.Schedule{WindowStart: "22:00", WindowEnd: "05:00", TimeZone: "Europe/Berlin"}
	// pinned clock: 12:00 (way outside)
	now := time.Date(2026, 4, 17, 12, 0, 0, 0, loc)
	var fires int32
	wrapped := wrapCountRunner(t, &fires)
	l := New(Options{
		Schedule: &sched,
		Runner:   wrapped,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Tick:     time.Nanosecond,
		Clock:    func() time.Time { return now },
	})
	l.tick(context.Background())
	if got := atomic.LoadInt32(&fires); got != 0 {
		t.Fatalf("fires = %d, want 0", got)
	}
}

// wrapCountRunner returns a Runner that increments *n every time
// TriggerNight is called. Since TriggerNight isn't an interface we
// wrap the real Runner and swap its provider for a fast mock; the
// counter sits in front of the provider via a helper adapter below.
func wrapCountRunner(t *testing.T, n *int32) *runner.Runner {
	t.Helper()
	fam := family.NewStore()
	fam.Seed([]family.Member{{
		Name: "rick", Role: "r", SystemPrompt: "p",
		Duties: []family.Duty{{Type: "lint-fix", Interval: "24h", Priority: family.PriorityHigh}},
	}})
	db, _ := storage.Open(context.Background(), ":memory:")
	t.Cleanup(func() { _ = db.Close() })
	r, _ := runner.New(runner.Deps{
		Family:   fam,
		Duties:   duty.NewBuiltinRegistry(),
		Storage:  db,
		Provider: &countProvider{n: n},
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	return r
}

// countProvider is a tiny Provider that increments a shared counter
// on every Run call — used as a proxy for "TriggerNight fired at
// least one dispatch".
type countProvider struct{ n *int32 }

func (c *countProvider) Name() string { return "count" }

func (c *countProvider) Run(ctx context.Context, req provider.Request) (*provider.Result, error) {
	atomic.AddInt32(c.n, 1)
	return &provider.Result{Summary: "ok"}, nil
}
