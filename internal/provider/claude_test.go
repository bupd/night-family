package provider

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// stubBin writes a tiny shell script to t.TempDir and returns its
// path. The script echoes whatever stdin handed it prefixed by a
// header, so composePrompt is observable in the test.
func stubBin(t *testing.T, script string) string {
	t.Helper()
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "claude")
	if err := writeExec(path, script); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	return path
}

// writeExec writes a shebang wrapper and marks it executable.
func writeExec(path, script string) error {
	const hdr = "#!/usr/bin/env sh\n"
	if err := writeFile(path, hdr+script); err != nil {
		return err
	}
	return exec.Command("chmod", "+x", path).Run()
}

func TestClaudeHappyPath(t *testing.T) {
	bin := stubBin(t, `cat <<EOF
summary line one
summary line two
EOF
`)
	c := &Claude{Bin: bin, Timeout: 5 * time.Second}
	res, err := c.Run(context.Background(), Request{
		Member:       "rick",
		MemberPrompt: "you are rick",
		Duty:         "vuln-scan",
		DutyPrompt:   "find bugs",
		RepoRoot:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Err != nil {
		t.Fatalf("res.Err = %v", res.Err)
	}
	if !strings.Contains(res.Summary, "summary line one") {
		t.Errorf("Summary = %q", res.Summary)
	}
}

func TestClaudeFailureExitCode(t *testing.T) {
	bin := stubBin(t, `echo "boom" >&2; exit 3`)
	c := &Claude{Bin: bin, Timeout: 5 * time.Second}
	res, _ := c.Run(context.Background(), Request{Member: "rick", Duty: "x"})
	if res == nil || res.Err == nil {
		t.Fatalf("expected res.Err, got res=%v", res)
	}
	if !strings.Contains(res.Err.Error(), "boom") {
		t.Errorf("err = %v", res.Err)
	}
}

func TestClaudeMissingBinary(t *testing.T) {
	c := &Claude{Bin: "definitely-not-a-real-binary-9f8e7d"}
	res, err := c.Run(context.Background(), Request{Member: "rick", Duty: "x"})
	if res == nil || res.Err == nil {
		t.Fatalf("expected res.Err when binary is missing, got res=%v err=%v", res, err)
	}
}

func TestComposePromptIncludesEssentials(t *testing.T) {
	out := composePrompt(Request{
		Member: "morty", MemberPrompt: "you are morty",
		Duty: "docs-drift", DutyPrompt: "fix docs",
		RepoRoot: "/tmp/repo",
		Args:     map[string]any{"foo": 1},
	})
	for _, want := range []string{"morty", "you are morty", "docs-drift", "fix docs", "/tmp/repo", "foo: 1"} {
		if !strings.Contains(out, want) {
			t.Errorf("composePrompt missing %q\n--- prompt ---\n%s", want, out)
		}
	}
}
