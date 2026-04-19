package duty

import (
	"testing"

	"github.com/bupd/night-family/internal/family"
)

func TestBuiltinsUniqueAndComplete(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range Builtins() {
		if seen[d.Type] {
			t.Errorf("duplicate built-in type: %s", d.Type)
		}
		seen[d.Type] = true
		if d.Description == "" {
			t.Errorf("%s: empty description", d.Type)
		}
		if !d.Builtin {
			t.Errorf("%s: Builtin flag not set", d.Type)
		}
		switch d.Output {
		case OutputPR, OutputIssue, OutputIssuePlusPR, OutputNote:
		default:
			t.Errorf("%s: invalid Output %q", d.Type, d.Output)
		}
		switch d.CostTier {
		case family.CostLow, family.CostMedium, family.CostHigh:
		default:
			t.Errorf("%s: invalid CostTier %q", d.Type, d.CostTier)
		}
		switch d.Risk {
		case family.RiskLow, family.RiskMedium, family.RiskHigh:
		default:
			t.Errorf("%s: invalid Risk %q", d.Type, d.Risk)
		}
	}
	// Spot-check: must cover the duty types referenced by the default roster.
	defaultDutyTypes := []string{
		"vuln-scan", "arch-review", "docs-drift", "release-notes",
		"readme-refresh", "changelog-groom", "test-coverage-gap",
		"flaky-test-detect", "dead-code", "refactor-hotspots",
		"lint-fix", "typo-fix", "dep-update-patch", "log-drift",
		"metric-coverage",
	}
	for _, want := range defaultDutyTypes {
		if !seen[want] {
			t.Errorf("built-in catalogue missing %q (referenced by default roster)", want)
		}
	}
}

func TestRegistryOperations(t *testing.T) {
	r := NewRegistry()
	if r.Len() != 0 {
		t.Fatalf("fresh Registry Len = %d", r.Len())
	}
	r.Register(Info{
		Type: "custom", Description: "x",
		Output: OutputNote, CostTier: family.CostLow, Risk: family.RiskLow,
	})
	if !r.Has("custom") {
		t.Errorf("Has(custom) false after Register")
	}
	if _, ok := r.Get("custom"); !ok {
		t.Errorf("Get(custom) missing")
	}
	if _, ok := r.Get("ghost"); ok {
		t.Errorf("Get(ghost) found an item")
	}
	if r.Len() != 1 {
		t.Errorf("Len = %d, want 1", r.Len())
	}
}

func TestBuiltinRegistryPopulated(t *testing.T) {
	r := NewBuiltinRegistry()
	if r.Len() != len(Builtins()) {
		t.Fatalf("Len = %d, want %d", r.Len(), len(Builtins()))
	}
}
