package storage

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestInsertGetListNight(t *testing.T) {
	db := openMem(t)
	ctx := context.Background()
	started := time.Date(2026, 4, 17, 22, 0, 0, 0, time.UTC)

	n := Night{
		ID:        "night_01HTEST000000000000000AAA",
		StartedAt: started,
		PlanJSON:  `{"slots":[{"member":"rick","duty":"vuln-scan"}]}`,
	}
	if err := db.InsertNight(ctx, n); err != nil {
		t.Fatalf("InsertNight: %v", err)
	}
	got, err := db.GetNight(ctx, n.ID)
	if err != nil {
		t.Fatalf("GetNight: %v", err)
	}
	if !got.StartedAt.Equal(started) {
		t.Errorf("StartedAt mismatch")
	}
	if got.PlanJSON != n.PlanJSON {
		t.Errorf("PlanJSON mismatch")
	}
	if _, err := db.GetNight(ctx, "night_missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get(missing) err = %v", err)
	}

	finished := started.Add(time.Hour)
	if err := db.FinishNight(ctx, n.ID, finished, "done, 1 PR opened"); err != nil {
		t.Fatalf("FinishNight: %v", err)
	}
	got, _ = db.GetNight(ctx, n.ID)
	if got.FinishedAt == nil || !got.FinishedAt.Equal(finished) {
		t.Errorf("FinishedAt = %v", got.FinishedAt)
	}
	if got.Summary == nil || *got.Summary != "done, 1 PR opened" {
		t.Errorf("Summary = %v", got.Summary)
	}

	list, err := db.ListNights(ctx, 10)
	if err != nil {
		t.Fatalf("ListNights: %v", err)
	}
	if len(list) != 1 || list[0].ID != n.ID {
		t.Errorf("list = %+v", list)
	}
}
