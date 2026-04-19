package planner

import (
	"testing"
	"time"

	"github.com/bupd/night-family/internal/duty"
	"github.com/bupd/night-family/internal/family"
	"github.com/bupd/night-family/internal/schedule"
)

func newTestInput(t *testing.T) Input {
	t.Helper()
	fam := family.NewStore()
	defaults, err := family.LoadDefaults()
	if err != nil {
		t.Fatalf("LoadDefaults: %v", err)
	}
	fam.Seed(defaults)
	sched := schedule.Default()
	return Input{
		Family:   fam,
		Duties:   duty.NewBuiltinRegistry(),
		Schedule: &sched,
		Now:      time.Date(2026, 4, 17, 23, 0, 0, 0, time.UTC),
	}
}

func TestPlanProducesSlots(t *testing.T) {
	in := newTestInput(t)
	plan, err := in.Plan()
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Slots) == 0 {
		t.Fatalf("no slots produced")
	}
	if plan.ReservedTokens <= 0 {
		t.Errorf("reserved tokens = %d", plan.ReservedTokens)
	}
	if plan.WindowEnd.Before(plan.WindowStart) {
		t.Errorf("window end before start")
	}
}

func TestPlanHonoursMemberCap(t *testing.T) {
	fam := family.NewStore()
	fam.Seed([]family.Member{
		{
			Name: "jerry", Role: "r", SystemPrompt: "p",
			CostTier: family.CostLow, RiskTolerance: family.RiskLow,
			MaxPRsPerNight: 1,
			Duties: []family.Duty{
				{Type: "lint-fix", Interval: "24h", Priority: family.PriorityHigh},
				{Type: "typo-fix", Interval: "48h", Priority: family.PriorityMedium},
				{Type: "dep-update-patch", Interval: "48h", Priority: family.PriorityMedium},
			},
		},
	})
	sched := schedule.Default()
	in := Input{
		Family:   fam,
		Duties:   duty.NewBuiltinRegistry(),
		Schedule: &sched,
		Now:      time.Date(2026, 4, 17, 23, 0, 0, 0, time.UTC),
	}
	plan, err := in.Plan()
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Slots) != 1 {
		t.Fatalf("slots = %d, want 1 (max_prs_per_night=1)", len(plan.Slots))
	}
	if len(plan.Skipped) != 2 {
		t.Fatalf("skipped = %d, want 2", len(plan.Skipped))
	}
	if plan.PerMemberCounts["jerry"] != 1 {
		t.Errorf("jerry count = %d, want 1", plan.PerMemberCounts["jerry"])
	}
}

func TestPlanHonoursBudget(t *testing.T) {
	in := newTestInput(t)
	in.BudgetTokens = 5000 // tiny — barely fits one cheap slot
	plan, err := in.Plan()
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if plan.ReservedTokens > 5000 {
		t.Errorf("reserved %d > budget 5000", plan.ReservedTokens)
	}
	if len(plan.Skipped) == 0 {
		t.Errorf("expected skipped slots when budget is tiny")
	}
}

func TestUnknownDutyGetsFlagged(t *testing.T) {
	fam := family.NewStore()
	fam.Seed([]family.Member{
		{
			Name: "squanchy", Role: "r", SystemPrompt: "p",
			CostTier: family.CostMedium, RiskTolerance: family.RiskMedium,
			MaxPRsPerNight: 2,
			Duties:         []family.Duty{{Type: "squanch-it", Interval: "24h", Priority: family.PriorityHigh}},
		},
	})
	sched := schedule.Default()
	in := Input{
		Family:   fam,
		Duties:   duty.NewBuiltinRegistry(),
		Schedule: &sched,
		Now:      time.Date(2026, 4, 17, 23, 0, 0, 0, time.UTC),
	}
	plan, err := in.Plan()
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Slots) != 1 {
		t.Fatalf("slots = %d, want 1", len(plan.Slots))
	}
	if plan.Slots[0].Reason == "" {
		t.Errorf("expected Reason to be flagged for unknown duty type")
	}
}

func TestMissingDepErrors(t *testing.T) {
	if _, err := (Input{}).Plan(); err == nil {
		t.Fatalf("expected error for empty Input")
	}
}

func TestPrioritySort(t *testing.T) {
	fam := family.NewStore()
	fam.Seed([]family.Member{
		{
			Name: "rick", Role: "r", SystemPrompt: "p",
			CostTier: family.CostMedium, RiskTolerance: family.RiskMedium,
			MaxPRsPerNight: 5,
			Duties: []family.Duty{
				{Type: "lint-fix", Interval: "24h", Priority: family.PriorityLow},
				{Type: "vuln-scan", Interval: "24h", Priority: family.PriorityHigh},
				{Type: "typo-fix", Interval: "48h", Priority: family.PriorityMedium},
			},
		},
	})
	sched := schedule.Default()
	in := Input{
		Family:   fam,
		Duties:   duty.NewBuiltinRegistry(),
		Schedule: &sched,
		Now:      time.Date(2026, 4, 17, 23, 0, 0, 0, time.UTC),
	}
	plan, err := in.Plan()
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Slots) != 3 {
		t.Fatalf("slots = %d, want 3", len(plan.Slots))
	}
	if plan.Slots[0].Duty != "vuln-scan" {
		t.Errorf("first slot duty = %q, want vuln-scan", plan.Slots[0].Duty)
	}
	if plan.Slots[2].Duty != "lint-fix" {
		t.Errorf("last slot duty = %q, want lint-fix", plan.Slots[2].Duty)
	}
}
