// Package provider abstracts over the LLM-backed runtime that actually
// does a duty's work. v1 ships with a single adapter — the noop/mock
// provider — so the rest of the system can be built and tested without
// burning real tokens. Real adapters (Claude Code, Codex) land in
// follow-up iterations.
package provider

import (
	"context"
	"time"
)

// Request is what the runner hands to a provider when dispatching a
// duty. Everything a provider needs to form its prompt is here: the
// member's system prompt, the duty type, the repository root, and
// anything extra the duty-specific planner attached as Args.
type Request struct {
	Member        string
	MemberPrompt  string
	Duty          string
	DutyPrompt    string
	RepoRoot      string
	Args          map[string]any
	EstimatedToks int
}

// Result is what a provider returns after a run completes (or fails).
// Token counts are optional — not every provider reports them.
type Result struct {
	Summary   string
	Branch    string
	PRURL     string
	TokensIn  int
	TokensOut int
	Err       error
}

// Provider is the seam a runner calls to execute one duty.
type Provider interface {
	Name() string
	Run(ctx context.Context, req Request) (*Result, error)
}

// SessionStatus is a best-effort snapshot of the provider's remaining
// capacity. Fields are independent — a provider may be able to fill
// some but not others. Zero values mean "unknown"; Confidence tells
// consumers how much to trust the numbers.
type SessionStatus struct {
	Provider                string
	RemainingTokensEstimate int
	WindowEndsAt            time.Time
	Confidence              string // "low" | "medium" | "high"
}

// StatusProber is an optional interface providers may implement.
// Absence = "I can't report session status." Runner + server degrade
// gracefully when a provider doesn't implement it.
type StatusProber interface {
	SessionStatus(ctx context.Context) (SessionStatus, error)
}
