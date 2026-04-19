package provider

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestCodexHappyPath(t *testing.T) {
	bin := stubBin(t, `cat <<EOF
codex-summary-line
EOF
`)
	c := &Codex{Bin: bin, Timeout: 5 * time.Second}
	res, err := c.Run(context.Background(), Request{Member: "rick", Duty: "vuln-scan", RepoRoot: t.TempDir()})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Err != nil {
		t.Fatalf("res.Err = %v", res.Err)
	}
	if !strings.Contains(res.Summary, "codex-summary-line") {
		t.Errorf("Summary = %q", res.Summary)
	}
}

func TestCodexMissingBinary(t *testing.T) {
	c := &Codex{Bin: "definitely-not-real-codex-8f7"}
	res, _ := c.Run(context.Background(), Request{Member: "rick", Duty: "x"})
	if res == nil || res.Err == nil {
		t.Fatalf("expected res.Err when binary is missing")
	}
}

func TestCodexName(t *testing.T) {
	if (&Codex{}).Name() != "codex" {
		t.Errorf("wrong name")
	}
}
