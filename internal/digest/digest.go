// Package digest renders a markdown summary of a completed night —
// the thing you actually read at 9am with coffee.
//
// Digests are deliberately static markdown, not HTML. They can be
// piped into email, pasted into Slack, committed into a log repo,
// or rendered by any markdown viewer.
package digest

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bupd/night-family/internal/storage"
)

// Night is the input bundle — a Night plus every Run that cites it.
type Night struct {
	storage.Night
	Runs []storage.Run
	PRs  []storage.PR
}

// Render returns the markdown body.
func Render(n Night) string {
	var b strings.Builder

	title := "Night " + n.ID
	if !n.StartedAt.IsZero() {
		title = n.StartedAt.Format("2006-01-02") + " — " + title
	}
	fmt.Fprintf(&b, "# %s\n\n", title)

	dur := "in progress"
	if n.FinishedAt != nil {
		dur = n.FinishedAt.Sub(n.StartedAt).Round(time.Second).String()
	}
	fmt.Fprintf(&b, "Duration: %s\n\n", dur)

	// Run status breakdown.
	byStatus := map[storage.RunStatus]int{}
	for _, r := range n.Runs {
		byStatus[r.Status]++
	}
	fmt.Fprintf(&b, "## Summary\n\n")
	fmt.Fprintf(&b, "- **Runs dispatched:** %d\n", len(n.Runs))
	for _, st := range []storage.RunStatus{storage.RunSucceeded, storage.RunFailed, storage.RunCancelled} {
		if c, ok := byStatus[st]; ok && c > 0 {
			fmt.Fprintf(&b, "  - %s: %d\n", st, c)
		}
	}
	if len(n.PRs) > 0 {
		fmt.Fprintf(&b, "- **PRs opened:** %d\n", len(n.PRs))
	}
	if n.Night.Summary != nil && *n.Night.Summary != "" {
		fmt.Fprintf(&b, "- Night note: %s\n", *n.Night.Summary)
	}
	fmt.Fprintf(&b, "\n")

	if len(n.PRs) > 0 {
		fmt.Fprintf(&b, "## PRs to review\n\n")
		prs := append([]storage.PR(nil), n.PRs...)
		sort.SliceStable(prs, func(i, j int) bool { return prs[i].OpenedAt.After(prs[j].OpenedAt) })
		for _, p := range prs {
			title := p.Member + " / " + p.Duty
			if p.Title != nil && *p.Title != "" {
				title = *p.Title
			}
			fmt.Fprintf(&b, "- [%s](%s) — `%s` / `%s`\n", title, p.URL, p.Member, p.Duty)
		}
		fmt.Fprintf(&b, "\n")
	}

	fmt.Fprintf(&b, "## Runs\n\n")
	if len(n.Runs) == 0 {
		fmt.Fprintf(&b, "_No runs._\n")
	} else {
		runs := append([]storage.Run(nil), n.Runs...)
		sort.SliceStable(runs, func(i, j int) bool { return runs[i].StartedAt.Before(runs[j].StartedAt) })
		for _, r := range runs {
			marker := "✓"
			if r.Status != storage.RunSucceeded {
				marker = "✗"
			}
			line := fmt.Sprintf("- %s `%s` / `%s` — %s", marker, r.Member, r.Duty, r.Status)
			if r.PRURL != nil && *r.PRURL != "" {
				line += fmt.Sprintf(" — [PR](%s)", *r.PRURL)
			}
			fmt.Fprintln(&b, line)
			if r.Summary != nil && *r.Summary != "" {
				sum := *r.Summary
				if len(sum) > 400 {
					sum = sum[:400] + "…"
				}
				fmt.Fprintf(&b, "    > %s\n", strings.ReplaceAll(sum, "\n", "\n    > "))
			}
			if r.Error != nil && *r.Error != "" {
				errMsg := *r.Error
				if len(errMsg) > 200 {
					errMsg = errMsg[:200] + "…"
				}
				fmt.Fprintf(&b, "    > error: %s\n", errMsg)
			}
		}
	}

	return b.String()
}
