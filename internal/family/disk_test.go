package family

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDiskDirMissingIsSilent(t *testing.T) {
	members, errs := LoadDiskDir(filepath.Join(t.TempDir(), "nope"))
	if len(errs) != 0 {
		t.Errorf("errs = %v, want none", errs)
	}
	if len(members) != 0 {
		t.Errorf("members = %v, want none", members)
	}
}

func TestLoadDiskDirPicksUpYAML(t *testing.T) {
	dir := t.TempDir()
	yaml := `name: custom-ops
role: Custom operations persona
system_prompt: |
  You are a custom operator.
duties:
  - type: lint-fix
    interval: 24h
    priority: high
risk_tolerance: low
cost_tier: low
`
	if err := os.WriteFile(filepath.Join(dir, "custom-ops.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	members, errs := LoadDiskDir(dir)
	if len(errs) != 0 {
		t.Fatalf("errs = %v", errs)
	}
	if len(members) != 1 {
		t.Fatalf("members = %d, want 1", len(members))
	}
	if members[0].Name != "custom-ops" {
		t.Errorf("name = %q", members[0].Name)
	}
}

func TestLoadDiskDirInvalidFilesFailFast(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "broken.yaml"), []byte("nope: [\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, errs := LoadDiskDir(dir)
	if len(errs) == 0 {
		t.Fatalf("expected errs for broken YAML")
	}
}
