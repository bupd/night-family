// Package runner dispatches a single (member, duty) pair through a
// provider, persisting the lifecycle to storage.
//
// Runs go queued → running → (succeeded | failed | cancelled). Every
// transition is recorded, so the API can report progress live.
package runner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bupd/night-family/internal/duty"
	"github.com/bupd/night-family/internal/family"
	"github.com/bupd/night-family/internal/provider"
	"github.com/bupd/night-family/internal/storage"
	"github.com/bupd/night-family/internal/ulid"
)

// Deps are the runtime dependencies a Runner needs.
type Deps struct {
	Family   *family.Store
	Duties   *duty.Registry
	Storage  *storage.DB
	Provider provider.Provider
	Logger   *slog.Logger
	// RepoRoot is the path passed to providers as the working dir.
	RepoRoot string
}

// Runner orchestrates one-off duty execution.
type Runner struct {
	deps Deps
}

// New returns a Runner with the given deps. All fields except
// RepoRoot must be non-nil.
func New(deps Deps) (*Runner, error) {
	if deps.Family == nil || deps.Duties == nil || deps.Storage == nil || deps.Provider == nil || deps.Logger == nil {
		return nil, fmt.Errorf("runner: all deps (Family/Duties/Storage/Provider/Logger) required")
	}
	return &Runner{deps: deps}, nil
}

// DispatchRequest is what callers hand to Dispatch.
type DispatchRequest struct {
	Member  string
	Duty    string
	NightID *string
	Args    map[string]any
}

// Dispatch creates a queued Run, transitions it to running, invokes
// the provider, and records the terminal state. Returns the final
// Run as stored.
func (r *Runner) Dispatch(ctx context.Context, req DispatchRequest) (storage.Run, error) {
	member, err := r.deps.Family.Get(req.Member)
	if err != nil {
		return storage.Run{}, fmt.Errorf("runner: family.Get %q: %w", req.Member, err)
	}
	info, known := r.deps.Duties.Get(req.Duty)
	if !known {
		// Prompt-only duty fallback: keep going with best-guess info.
		info = duty.Info{Type: req.Duty, Output: duty.OutputNote, CostTier: family.CostMedium, Risk: family.RiskMedium}
	}

	now := time.Now().UTC()
	run := storage.Run{
		ID:        ulid.Make("run"),
		NightID:   req.NightID,
		Member:    member.Name,
		Duty:      req.Duty,
		Status:    storage.RunQueued,
		StartedAt: now,
	}
	if err := r.deps.Storage.InsertRun(ctx, run); err != nil {
		return storage.Run{}, fmt.Errorf("runner: insert: %w", err)
	}

	// Transition to running.
	if err := r.deps.Storage.UpdateRunStatus(ctx, run.ID, storage.RunRunning, nil, nil, nil, nil, nil); err != nil {
		return storage.Run{}, fmt.Errorf("runner: mark running: %w", err)
	}
	r.deps.Logger.Info("run dispatched", "run", run.ID, "member", run.Member, "duty", run.Duty)

	pReq := provider.Request{
		Member:       member.Name,
		MemberPrompt: member.SystemPrompt,
		Duty:         req.Duty,
		DutyPrompt:   info.Description,
		RepoRoot:     r.deps.RepoRoot,
		Args:         req.Args,
	}
	res, callErr := r.deps.Provider.Run(ctx, pReq)

	finished := time.Now().UTC()
	status := storage.RunSucceeded
	var errMsg *string
	var summary *string
	var tokensIn, tokensOut *int

	if callErr != nil {
		status = storage.RunFailed
		msg := callErr.Error()
		errMsg = &msg
	} else if res != nil && res.Err != nil {
		status = storage.RunFailed
		msg := res.Err.Error()
		errMsg = &msg
	} else if res != nil {
		if res.Summary != "" {
			s := res.Summary
			summary = &s
		}
		if res.TokensIn > 0 {
			v := res.TokensIn
			tokensIn = &v
		}
		if res.TokensOut > 0 {
			v := res.TokensOut
			tokensOut = &v
		}
	}

	if err := r.deps.Storage.UpdateRunStatus(ctx, run.ID, status, &finished, tokensIn, tokensOut, summary, errMsg); err != nil {
		return storage.Run{}, fmt.Errorf("runner: mark terminal: %w", err)
	}
	r.deps.Logger.Info("run finished",
		"run", run.ID, "status", status,
		"tokens_in", deref(tokensIn), "tokens_out", deref(tokensOut))

	final, err := r.deps.Storage.GetRun(ctx, run.ID)
	if err != nil {
		return storage.Run{}, fmt.Errorf("runner: reload: %w", err)
	}
	return final, nil
}

func deref(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
