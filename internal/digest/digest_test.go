package digest

import (
	"strings"
	"testing"
	"time"

	"github.com/bupd/night-family/internal/storage"
)

func TestRenderIncludesExpectedSections(t *testing.T) {
	started := time.Date(2026, 4, 17, 22, 0, 0, 0, time.UTC)
	finished := started.Add(90 * time.Minute)
	summary := "all-good summary"
	prURL := "https://github.com/foo/bar/pull/42"

	n := Night{
		Night: storage.Night{
			ID:         "night_01HTEST0000000000000000AA",
			StartedAt:  started,
			FinishedAt: &finished,
			Summary:    stringPtr("night note"),
		},
		Runs: []storage.Run{
			{
				ID: "run_1", Member: "jerry", Duty: "lint-fix",
				Status:    storage.RunSucceeded,
				StartedAt: started,
				Summary:   &summary,
				PRURL:     &prURL,
			},
			{
				ID: "run_2", Member: "rick", Duty: "vuln-scan",
				Status:    storage.RunFailed,
				StartedAt: started.Add(time.Minute),
				Error:     stringPtr("bang"),
			},
		},
		PRs: []storage.PR{
			{
				ID: "pr_1", URL: prURL, Member: "jerry", Duty: "lint-fix",
				Title:    stringPtr("docs: fix typo"),
				OpenedAt: started,
				State:    storage.PROpen,
			},
		},
	}

	out := Render(n)
	for _, want := range []string{
		"# 2026-04-17 — Night night_01HTEST",
		"Duration: 1h30m0s",
		"**Runs dispatched:** 2",
		"succeeded: 1",
		"failed: 1",
		"**PRs opened:** 1",
		"PRs to review",
		"`jerry` / `lint-fix`",
		"all-good summary",
		"vuln-scan",
		"error: bang",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("digest missing %q\n--- output ---\n%s", want, out)
		}
	}
}

func TestRenderHandlesEmptyNight(t *testing.T) {
	out := Render(Night{
		Night: storage.Night{
			ID:        "night_empty",
			StartedAt: time.Now().UTC(),
		},
	})
	if !strings.Contains(out, "No runs.") {
		t.Errorf("empty digest missing fallback: %s", out)
	}
}

func stringPtr(s string) *string { return &s }
