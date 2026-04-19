package runner

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/bupd/night-family/internal/duty"
	"github.com/bupd/night-family/internal/family"
	"github.com/bupd/night-family/internal/provider"
	"github.com/bupd/night-family/internal/storage"
)

func newTestRunner(t *testing.T, p provider.Provider) (*Runner, *storage.DB) {
	t.Helper()
	fam := family.NewStore()
	defaults, err := family.LoadDefaults()
	if err != nil {
		t.Fatalf("LoadDefaults: %v", err)
	}
	fam.Seed(defaults)
	db, err := storage.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	r, err := New(Deps{
		Family:   fam,
		Duties:   duty.NewBuiltinRegistry(),
		Storage:  db,
		Provider: p,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return r, db
}

func TestDispatchSuccess(t *testing.T) {
	mock := provider.NewMock()
	r, db := newTestRunner(t, mock)
	run, err := r.Dispatch(context.Background(), DispatchRequest{
		Member: "jerry", Duty: "lint-fix",
	})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if run.Status != storage.RunSucceeded {
		t.Errorf("status = %q, want succeeded", run.Status)
	}
	if run.Summary == nil || *run.Summary == "" {
		t.Errorf("summary missing")
	}
	if run.TokensIn == nil || *run.TokensIn != mock.TokensIn {
		t.Errorf("tokens_in = %v", run.TokensIn)
	}
	got, err := db.GetRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.Status != storage.RunSucceeded {
		t.Errorf("persisted status = %q", got.Status)
	}
}

func TestDispatchUnknownMember(t *testing.T) {
	r, _ := newTestRunner(t, provider.NewMock())
	_, err := r.Dispatch(context.Background(), DispatchRequest{Member: "ghost", Duty: "lint-fix"})
	if err == nil {
		t.Fatalf("expected error for unknown member")
	}
}

func TestDispatchRecordsFailure(t *testing.T) {
	mock := provider.NewMock()
	mock.FailMember = "jerry"
	r, db := newTestRunner(t, mock)
	run, err := r.Dispatch(context.Background(), DispatchRequest{Member: "jerry", Duty: "lint-fix"})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if run.Status != storage.RunFailed {
		t.Errorf("status = %q, want failed", run.Status)
	}
	if run.Error == nil || *run.Error == "" {
		t.Errorf("error message missing")
	}
	persisted, _ := db.GetRun(context.Background(), run.ID)
	if persisted.Status != storage.RunFailed {
		t.Errorf("persisted status = %q", persisted.Status)
	}
}

func TestDispatchCtxCancelledSurfaces(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before dispatch
	r, _ := newTestRunner(t, provider.NewMock())
	run, err := r.Dispatch(ctx, DispatchRequest{Member: "jerry", Duty: "lint-fix"})
	// A cancelled context may either fail the INSERT (returning an
	// error + empty Run) or make it through to a failed Run record —
	// both are valid outcomes; we only want to be sure we don't
	// silently succeed.
	if err == nil && run.Status == storage.RunSucceeded {
		t.Errorf("run succeeded despite cancelled context")
	}
	if err != nil && !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "context canceled") {
		t.Logf("non-cancel err (acceptable): %v", err)
	}
}
