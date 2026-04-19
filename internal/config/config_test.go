package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingIsEmpty(t *testing.T) {
	d, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if d.Addr != "" || d.Provider != "" || len(d.ClaudeArgs) != 0 {
		t.Errorf("missing file produced non-zero Daemon: %+v", d)
	}
}

func TestLoadParses(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	raw := `
addr: 127.0.0.1:8001
log_level: debug
provider: claude
claude_args:
  - --dangerously-skip-permissions
reviewers:
  - coderabbitai
  - cubic-dev-ai
auto_trigger: true
`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	d, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if d.Addr != "127.0.0.1:8001" {
		t.Errorf("addr = %q", d.Addr)
	}
	if d.Provider != "claude" {
		t.Errorf("provider = %q", d.Provider)
	}
	if len(d.ClaudeArgs) != 1 || d.ClaudeArgs[0] != "--dangerously-skip-permissions" {
		t.Errorf("claude_args = %v", d.ClaudeArgs)
	}
	if len(d.Reviewers) != 2 {
		t.Errorf("reviewers = %v", d.Reviewers)
	}
	if !d.AutoTrigger {
		t.Errorf("auto_trigger not set")
	}
}

func TestLoadRejectsInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	_ = os.WriteFile(path, []byte("addr: [\n"), 0o644)
	if _, err := Load(path); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestDefaultPathHonoursXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-fake")
	if got := DefaultPath(); got != "/tmp/xdg-fake/night-family/config.yaml" {
		t.Errorf("path = %q", got)
	}
}
