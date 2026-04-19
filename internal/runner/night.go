package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bupd/night-family/internal/planner"
	"github.com/bupd/night-family/internal/schedule"
	"github.com/bupd/night-family/internal/storage"
	"github.com/bupd/night-family/internal/ulid"
)

// NightOptions shape the TriggerNight call.
type NightOptions struct {
	// OnlyMembers/OnlyDuties, when non-empty, restrict the plan to
	// those filters.
	OnlyMembers []string
	OnlyDuties  []string
	// DryRun plans but does not dispatch runs.
	DryRun bool
	// Budget caps total token spend across the night. Zero = no cap.
	Budget int
}

// NightResult is what TriggerNight returns. Errors inside individual
// runs do not stop the night — they're recorded against each Run.
type NightResult struct {
	ID      string        `json:"id"`
	Runs    []storage.Run `json:"runs"`
	Plan    planner.Plan  `json:"plan"`
	Skipped int           `json:"skipped"`
}

// TriggerNight plans + dispatches a night. v1 runs slots sequentially
// so token budgets are respected deterministically.
func (r *Runner) TriggerNight(ctx context.Context, sched *schedule.Schedule, opts NightOptions) (NightResult, error) {
	plan, err := planner.Input{
		Family:       r.deps.Family,
		Duties:       r.deps.Duties,
		Schedule:     sched,
		Now:          time.Now(),
		BudgetTokens: opts.Budget,
	}.Plan()
	if err != nil {
		return NightResult{}, fmt.Errorf("runner: plan: %w", err)
	}
	plan.Slots = filterSlots(plan.Slots, opts.OnlyMembers, opts.OnlyDuties)

	nightID := ulid.Make("night")
	started := time.Now().UTC()

	planJSON, _ := json.Marshal(plan)
	if err := r.deps.Storage.InsertNight(ctx, storage.Night{
		ID:        nightID,
		StartedAt: started,
		PlanJSON:  string(planJSON),
	}); err != nil {
		return NightResult{}, fmt.Errorf("runner: insert night: %w", err)
	}

	// Record a budget snapshot at night start so the dashboard /
	// /api/v1/budget can show "we reserved X of an estimated Y
	// remaining" without extra probing. The remaining estimate is
	// best-effort; once a session-status probe lands this becomes a
	// real observation.
	_, _ = r.deps.Storage.InsertBudgetSnapshot(ctx, storage.BudgetSnapshot{
		Provider:                r.deps.Provider.Name(),
		RemainingTokensEstimate: max(plan.BudgetTokens, plan.ReservedTokens*2),
		ReservedForTonight:      plan.ReservedTokens,
		WindowEndsAt:            &plan.WindowEnd,
		Confidence:              "low",
	})

	var runs []storage.Run
	if !opts.DryRun {
		for _, slot := range plan.Slots {
			select {
			case <-ctx.Done():
				r.deps.Logger.Warn("night cancelled mid-dispatch", "night", nightID)
				goto finish
			default:
			}
			nightRef := nightID
			run, err := r.Dispatch(ctx, DispatchRequest{
				Member:  slot.Member,
				Duty:    slot.Duty,
				NightID: &nightRef,
			})
			if err != nil {
				r.deps.Logger.Error("dispatch failed", "night", nightID, "err", err)
				continue
			}
			runs = append(runs, run)
		}
	}
finish:
	summary := fmt.Sprintf("night %s: %d runs dispatched (%d skipped at plan time)",
		nightID, len(runs), len(plan.Skipped))
	_ = r.deps.Storage.FinishNight(ctx, nightID, time.Now().UTC(), summary)
	r.deps.Logger.Info("night done", "night", nightID, "runs", len(runs))

	return NightResult{
		ID:      nightID,
		Runs:    runs,
		Plan:    plan,
		Skipped: len(plan.Skipped),
	}, nil
}

// filterSlots returns only slots that match the allow-lists (or all
// slots when both lists are empty).
func filterSlots(in []planner.Slot, onlyMembers, onlyDuties []string) []planner.Slot {
	if len(onlyMembers) == 0 && len(onlyDuties) == 0 {
		return in
	}
	mem := toSet(onlyMembers)
	dut := toSet(onlyDuties)
	out := in[:0]
	for _, s := range in {
		if len(mem) > 0 && !mem[s.Member] {
			continue
		}
		if len(dut) > 0 && !dut[s.Duty] {
			continue
		}
		out = append(out, s)
	}
	return out
}

func toSet(xs []string) map[string]bool {
	if len(xs) == 0 {
		return nil
	}
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}
