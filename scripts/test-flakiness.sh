#!/usr/bin/env bash
# test-flakiness.sh — Run tests N times and report flaky tests.
#
# Usage: scripts/test-flakiness.sh [COUNT]
#   COUNT  Number of runs (default: 5)
#
# Output: JSON report of tests with inconsistent results.

set -euo pipefail

COUNT="${1:-5}"
PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "=== Flakiness analyzer: running tests ${COUNT} times ===" >&2

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

# Run tests COUNT times, collecting JSON output.
for i in $(seq 1 "$COUNT"); do
    echo "  run ${i}/${COUNT}..." >&2
    go test -race -json -count=1 ./... 2>/dev/null > "${tmpdir}/run_${i}.json" || true
done

# Analyze results with an inline Go program.
cat > "${tmpdir}/analyze.go" << 'GOEOF'
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type TestEvent struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Elapsed float64 `json:"Elapsed"`
}

type TestResult struct {
	Pass int
	Fail int
	Skip int
	Durations []float64
}

func main() {
	dir := os.Args[1]
	count := os.Args[2]

	results := map[string]*TestResult{} // key: "pkg::test"

	entries, _ := filepath.Glob(filepath.Join(dir, "run_*.json"))
	for _, path := range entries {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		dec := json.NewDecoder(f)
		for {
			var ev TestEvent
			if err := dec.Decode(&ev); err == io.EOF {
				break
			} else if err != nil {
				continue
			}
			if ev.Test == "" {
				continue
			}
			if ev.Action != "pass" && ev.Action != "fail" && ev.Action != "skip" {
				continue
			}
			key := ev.Package + "::" + ev.Test
			if results[key] == nil {
				results[key] = &TestResult{}
			}
			r := results[key]
			switch ev.Action {
			case "pass":
				r.Pass++
				r.Durations = append(r.Durations, ev.Elapsed)
			case "fail":
				r.Fail++
				r.Durations = append(r.Durations, ev.Elapsed)
			case "skip":
				r.Skip++
			}
		}
		f.Close()
	}

	// Find flaky tests (both pass and fail).
	type FlakyTest struct {
		Name     string    `json:"name"`
		Pass     int       `json:"pass"`
		Fail     int       `json:"fail"`
		Skip     int       `json:"skip,omitempty"`
		Rate     string    `json:"pass_rate"`
		Durations []float64 `json:"durations_sec"`
	}
	var flaky []FlakyTest
	for key, r := range results {
		if r.Pass > 0 && r.Fail > 0 {
			rate := fmt.Sprintf("%.0f%%", float64(r.Pass)/float64(r.Pass+r.Fail)*100)
			flaky = append(flaky, FlakyTest{
				Name:     key,
				Pass:     r.Pass,
				Fail:     r.Fail,
				Skip:     r.Skip,
				Rate:     rate,
				Durations: r.Durations,
			})
		}
	}
	sort.Slice(flaky, func(i, j int) bool {
		return flaky[i].Name < flaky[j].Name
	})

	totalTests := len(results)
	report := map[string]any{
		"runs":        count,
		"total_tests": totalTests,
		"flaky_count": len(flaky),
		"flaky_tests": flaky,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(report)

	if len(flaky) > 0 {
		fmt.Fprintf(os.Stderr, "\n⚠ Found %d flaky test(s) out of %d\n", len(flaky), totalTests)
		for _, f := range flaky {
			parts := strings.SplitN(f.Name, "::", 2)
			fmt.Fprintf(os.Stderr, "  %s (pass=%d fail=%d rate=%s)\n", parts[1], f.Pass, f.Fail, f.Rate)
		}
		os.Exit(1)
	} else {
		fmt.Fprintf(os.Stderr, "\n✓ No flaky tests found (%d tests, %s runs)\n", totalTests, count)
	}
}
GOEOF

cd "$PROJECT_ROOT"
go run "${tmpdir}/analyze.go" "$tmpdir" "$COUNT"
