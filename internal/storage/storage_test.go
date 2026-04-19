package storage

import (
	"context"
	"errors"
	"testing"
	"time"
)

func openMem(t *testing.T) *DB {
	t.Helper()
	db, err := Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestOpenAppliesMigrations(t *testing.T) {
	db := openMem(t)
	v, err := db.Version(context.Background())
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if v != "0001" {
		t.Fatalf("version = %q, want 0001", v)
	}
}

func TestInsertAndGetRun(t *testing.T) {
	db := openMem(t)
	ctx := context.Background()
	started := time.Date(2026, 4, 17, 22, 0, 0, 0, time.UTC)

	r := Run{
		ID:        "run_01HX000000000000000000TEST",
		Member:    "rick",
		Duty:      "vuln-scan",
		Status:    RunQueued,
		StartedAt: started,
	}
	if err := db.InsertRun(ctx, r); err != nil {
		t.Fatalf("InsertRun: %v", err)
	}
	got, err := db.GetRun(ctx, r.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.Member != "rick" || got.Duty != "vuln-scan" || got.Status != RunQueued {
		t.Errorf("round-trip mismatch: %+v", got)
	}
	if !got.StartedAt.Equal(started) {
		t.Errorf("StartedAt mismatch: %v vs %v", got.StartedAt, started)
	}

	if _, err := db.GetRun(ctx, "run_missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetRun(missing) err = %v, want ErrNotFound", err)
	}
}

func TestUpdateRunStatus(t *testing.T) {
	db := openMem(t)
	ctx := context.Background()
	started := time.Date(2026, 4, 17, 22, 0, 0, 0, time.UTC)
	r := Run{
		ID:        "run_01HX000000000000000000FOO",
		Member:    "jerry",
		Duty:      "lint-fix",
		Status:    RunRunning,
		StartedAt: started,
	}
	if err := db.InsertRun(ctx, r); err != nil {
		t.Fatalf("InsertRun: %v", err)
	}
	finished := started.Add(5 * time.Minute)
	in, out := 1000, 200
	summary := "done"
	if err := db.UpdateRunStatus(ctx, r.ID, RunSucceeded, &finished, &in, &out, &summary, nil); err != nil {
		t.Fatalf("UpdateRunStatus: %v", err)
	}
	got, err := db.GetRun(ctx, r.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.Status != RunSucceeded {
		t.Errorf("status = %q, want succeeded", got.Status)
	}
	if got.FinishedAt == nil || !got.FinishedAt.Equal(finished) {
		t.Errorf("FinishedAt = %v, want %v", got.FinishedAt, finished)
	}
	if got.TokensIn == nil || *got.TokensIn != 1000 {
		t.Errorf("TokensIn = %v", got.TokensIn)
	}
	if got.Summary == nil || *got.Summary != "done" {
		t.Errorf("Summary = %v", got.Summary)
	}
}

func TestListRunsFilters(t *testing.T) {
	db := openMem(t)
	ctx := context.Background()
	base := time.Date(2026, 4, 17, 22, 0, 0, 0, time.UTC)

	runs := []Run{
		{ID: "run_A", Member: "rick", Duty: "vuln-scan", Status: RunSucceeded, StartedAt: base.Add(1 * time.Minute)},
		{ID: "run_B", Member: "morty", Duty: "docs-drift", Status: RunSucceeded, StartedAt: base.Add(2 * time.Minute)},
		{ID: "run_C", Member: "rick", Duty: "arch-review", Status: RunFailed, StartedAt: base.Add(3 * time.Minute)},
	}
	for _, r := range runs {
		if err := db.InsertRun(ctx, r); err != nil {
			t.Fatalf("insert %s: %v", r.ID, err)
		}
	}

	all, err := db.ListRuns(ctx, ListRunsFilter{})
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("all count = %d", len(all))
	}
	if all[0].ID != "run_C" {
		t.Errorf("ordering: first = %q, want run_C (most recent)", all[0].ID)
	}

	onlyRick, err := db.ListRuns(ctx, ListRunsFilter{Member: "rick"})
	if err != nil {
		t.Fatalf("ListRuns(rick): %v", err)
	}
	if len(onlyRick) != 2 {
		t.Errorf("rick count = %d, want 2", len(onlyRick))
	}

	failed, err := db.ListRuns(ctx, ListRunsFilter{Status: RunFailed})
	if err != nil {
		t.Fatalf("ListRuns(failed): %v", err)
	}
	if len(failed) != 1 || failed[0].ID != "run_C" {
		t.Errorf("failed filter = %+v", failed)
	}
}
