// Package e2e exercises the full nfd binary via a child process.
//
// What's tested:
//   - binary builds (`go build` happens in TestMain).
//   - nfd boots with an in-memory DB, serves /healthz, /readyz, /version.
//   - POST /api/v1/nights/trigger against the default roster with the
//     mock provider produces 12 runs and records the Night.
//   - /metrics reflects the new counts.
//
// Skipped automatically when `go` isn't on PATH (shouldn't happen in
// CI but saves maintainers running from a stripped container).
package e2e

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var nfdBin string

func TestMain(m *testing.M) {
	// Build nfd once, up-front. Covers syntax/type regressions before
	// any test even runs.
	if _, err := exec.LookPath("go"); err != nil {
		fmt.Fprintln(os.Stderr, "e2e: 'go' not found on PATH; skipping")
		os.Exit(0)
	}
	dir, err := os.MkdirTemp("", "nfd-e2e-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "e2e: mktemp:", err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)

	nfdBin = filepath.Join(dir, "nfd")
	cmd := exec.Command("go", "build", "-o", nfdBin, "./cmd/nfd")
	cmd.Dir = projectRoot()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "e2e: build nfd:", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// projectRoot walks up until we find a go.mod. Keeps the test file
// location-independent.
func projectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "" // won't happen in a normal checkout
		}
		dir = parent
	}
}

// freePort asks the OS for an unused port by binding to :0.
func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return fmt.Sprintf("%d", port)
}

// startNfd launches the built binary with an in-memory DB on a
// dynamically allocated port. Returns (baseURL, stop).
func startNfd(t *testing.T) (string, func()) {
	t.Helper()
	port := freePort(t)
	addr := "127.0.0.1:" + port
	cmd := exec.Command(nfdBin, "-db", ":memory:", "-addr", addr, "-log-level", "error")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start nfd: %v", err)
	}
	base := "http://" + addr
	waitForReady(t, base)
	return base, func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}

// waitForReady polls /healthz with exponential backoff until 200 or timeout.
func waitForReady(t *testing.T, base string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	backoff := 25 * time.Millisecond
	const maxBackoff = 500 * time.Millisecond
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return
			}
		}
		time.Sleep(backoff)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
	t.Fatalf("daemon never became ready at %s", base)
}

func TestE2E_FullNightThroughMock(t *testing.T) {
	base, stop := startNfd(t)
	t.Cleanup(stop)

	resp, err := http.Post(base+"/api/v1/nights/trigger",
		"application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("trigger: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var res struct {
		ID   string           `json:"id"`
		Runs []map[string]any `json:"runs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.HasPrefix(res.ID, "night_") {
		t.Errorf("night id = %q", res.ID)
	}
	if len(res.Runs) < 10 {
		t.Errorf("runs dispatched = %d, want >= 10", len(res.Runs))
	}

	runsResp, err := http.Get(base + "/api/v1/runs")
	if err != nil {
		t.Fatalf("runs: %v", err)
	}
	defer runsResp.Body.Close()
	var runs struct {
		Items []map[string]any `json:"items"`
	}
	_ = json.NewDecoder(runsResp.Body).Decode(&runs)
	if len(runs.Items) != len(res.Runs) {
		t.Errorf("runs persisted = %d, dispatched = %d", len(runs.Items), len(res.Runs))
	}

	// /metrics should reflect the counts.
	mResp, err := http.Get(base + "/metrics")
	if err != nil {
		t.Fatalf("metrics: %v", err)
	}
	defer mResp.Body.Close()
	buf := make([]byte, 8192)
	n, _ := mResp.Body.Read(buf)
	body := string(buf[:n])
	wantRuns := fmt.Sprintf("nf_runs_total %d", len(res.Runs))
	if !strings.Contains(body, wantRuns) {
		t.Errorf("/metrics missing %q\n--- body ---\n%s", wantRuns, body)
	}
}

func TestE2E_HealthAndVersion(t *testing.T) {
	base, stop := startNfd(t)
	t.Cleanup(stop)

	for _, path := range []string{"/healthz", "/readyz", "/version", "/openapi.yaml", "/api/v1/family"} {
		resp, err := http.Get(base + path)
		if err != nil {
			t.Errorf("%s: %v", path, err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			t.Errorf("%s → %d", path, resp.StatusCode)
		}
	}
}
