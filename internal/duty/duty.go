// Package duty is the central registry of duty types known to
// night-family. Each entry captures the cheap static metadata a
// planner needs to reason about cost, risk, and output kind.
//
// The actual execution of a duty (prompt synthesis, provider call,
// output parsing, PR/issue creation) happens in a separate package.
// This package is side-effect-free — import it from the validator
// and the API without pulling in a provider dependency.
package duty

import (
	"sort"
	"sync"

	"github.com/bupd/night-family/internal/family"
)

// Output describes what a duty produces when it runs.
type Output string

const (
	OutputPR          Output = "pr"
	OutputIssue       Output = "issue"
	OutputIssuePlusPR Output = "issue-plus-pr"
	OutputNote        Output = "note"
)

// Info is the static, serialisable metadata for a duty type.
type Info struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Output      Output      `json:"output"`
	CostTier    family.Cost `json:"cost_tier"`
	Risk        family.Risk `json:"risk"`
	Builtin     bool        `json:"builtin"`
}

// Registry is a name → Info lookup. Safe for concurrent reads after
// Register has finished at startup.
type Registry struct {
	mu    sync.RWMutex
	items map[string]Info
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{items: make(map[string]Info)}
}

// Register adds or replaces a duty. Returns the stored Info.
func (r *Registry) Register(info Info) Info {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[info.Type] = info
	return info
}

// Get returns the Info for a type, or (zero, false) when unknown.
func (r *Registry) Get(typ string) (Info, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	i, ok := r.items[typ]
	return i, ok
}

// Has reports whether a duty type is known.
func (r *Registry) Has(typ string) bool {
	_, ok := r.Get(typ)
	return ok
}

// List returns all duties sorted by Type.
func (r *Registry) List() []Info {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Info, 0, len(r.items))
	for _, i := range r.items {
		out = append(out, i)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Type < out[j].Type })
	return out
}

// Len returns the number of registered duties.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.items)
}

// Builtins returns the canonical v1 duty catalogue. See docs/DUTIES.md
// for the narrative version. This is the single source of truth for
// what duty types ship with night-family.
func Builtins() []Info {
	return []Info{
		{
			Type:        "docs-drift",
			Description: "Find stale docs referring to renamed/removed code and fix them.",
			Output:      OutputPR,
			CostTier:    family.CostMedium,
			Risk:        family.RiskLow,
			Builtin:     true,
		},
		{
			Type:        "release-notes",
			Description: "Draft release notes from git log since the last tag.",
			Output:      OutputPR,
			CostTier:    family.CostLow,
			Risk:        family.RiskLow,
			Builtin:     true,
		},
		{
			Type:        "readme-refresh",
			Description: "Update READMEs when code or behaviour has drifted.",
			Output:      OutputPR,
			CostTier:    family.CostLow,
			Risk:        family.RiskLow,
			Builtin:     true,
		},
		{
			Type:        "changelog-groom",
			Description: "Normalise CHANGELOG entries for consistency.",
			Output:      OutputPR,
			CostTier:    family.CostLow,
			Risk:        family.RiskLow,
			Builtin:     true,
		},
		{
			Type:        "test-coverage-gap",
			Description: "Add unit tests for low-coverage exported functions.",
			Output:      OutputPR,
			CostTier:    family.CostHigh,
			Risk:        family.RiskMedium,
			Builtin:     true,
		},
		{
			Type:        "flaky-test-detect",
			Description: "Analyse CI history, flag flaky tests.",
			Output:      OutputIssue,
			CostTier:    family.CostMedium,
			Risk:        family.RiskLow,
			Builtin:     true,
		},
		{
			Type:        "vuln-scan",
			Description: "Dependency audit plus code-pattern scan.",
			Output:      OutputIssue,
			CostTier:    family.CostMedium,
			Risk:        family.RiskLow,
			Builtin:     true,
		},
		{
			Type:        "dead-code",
			Description: "Remove unused exports (PR requires human merge).",
			Output:      OutputPR,
			CostTier:    family.CostMedium,
			Risk:        family.RiskMedium,
			Builtin:     true,
		},
		{
			Type:        "refactor-hotspots",
			Description: "Identify high-churn, high-complexity files.",
			Output:      OutputIssue,
			CostTier:    family.CostHigh,
			Risk:        family.RiskLow,
			Builtin:     true,
		},
		{
			Type:        "lint-fix",
			Description: "Run the formatter + linter's auto-fixes.",
			Output:      OutputPR,
			CostTier:    family.CostLow,
			Risk:        family.RiskLow,
			Builtin:     true,
		},
		{
			Type:        "typo-fix",
			Description: "Fix typos in comments, docs, and safe identifiers.",
			Output:      OutputPR,
			CostTier:    family.CostLow,
			Risk:        family.RiskLow,
			Builtin:     true,
		},
		{
			Type:        "dep-update-patch",
			Description: "Bump patch-level dependencies one at a time.",
			Output:      OutputPR,
			CostTier:    family.CostLow,
			Risk:        family.RiskLow,
			Builtin:     true,
		},
		{
			Type:        "dep-update-minor",
			Description: "Bump minor-level dependencies, one per PR.",
			Output:      OutputPR,
			CostTier:    family.CostMedium,
			Risk:        family.RiskMedium,
			Builtin:     true,
		},
		{
			Type:        "todo-triage",
			Description: "Convert TODO comments into tracked GitHub issues.",
			Output:      OutputIssuePlusPR,
			CostTier:    family.CostMedium,
			Risk:        family.RiskLow,
			Builtin:     true,
		},
		{
			Type:        "arch-review",
			Description: "Long-form architectural commentary filed as an issue.",
			Output:      OutputIssue,
			CostTier:    family.CostHigh,
			Risk:        family.RiskLow,
			Builtin:     true,
		},
		{
			Type:        "log-drift",
			Description: "Find log lines referencing removed or renamed fields.",
			Output:      OutputIssue,
			CostTier:    family.CostLow,
			Risk:        family.RiskLow,
			Builtin:     true,
		},
		{
			Type:        "metric-coverage",
			Description: "Flag endpoints without metrics or traces.",
			Output:      OutputIssue,
			CostTier:    family.CostLow,
			Risk:        family.RiskLow,
			Builtin:     true,
		},
		{
			Type:        "ci-signal-noise",
			Description: "Spot noisy CI jobs or flaky notifications.",
			Output:      OutputIssue,
			CostTier:    family.CostMedium,
			Risk:        family.RiskLow,
			Builtin:     true,
		},
	}
}

// NewBuiltinRegistry returns a Registry pre-populated with the built-in
// catalogue.
func NewBuiltinRegistry() *Registry {
	r := NewRegistry()
	for _, d := range Builtins() {
		r.Register(d)
	}
	return r
}
