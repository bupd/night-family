package runner

import (
	"context"
	"io"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bupd/night-family/internal/duty"
	"github.com/bupd/night-family/internal/family"
	"github.com/bupd/night-family/internal/gitops"
	"github.com/bupd/night-family/internal/provider"
	"github.com/bupd/night-family/internal/storage"
)

// newRepo mirrors the helper in gitops_test.go — we can't import from
// a _test.go file, so we inline the minimal bootstrap we need.
func newRepoForRunner(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	base := t.TempDir()
	bare := filepath.Join(base, "origin.git")
	clone := filepath.Join(base, "clone")
	run := func(dir string, args ...string) {
		cmd := exec.Command("git", args...)
		if dir != "" {
			cmd.Dir = dir
		}
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	run("", "init", "--bare", bare)
	run("", "clone", bare, clone)
	run(clone, "config", "user.email", "test@example.com")
	run(clone, "config", "user.name", "Test User")
	run(clone, "commit", "--allow-empty", "-m", "initial")
	run(clone, "branch", "-M", "main")
	run(clone, "push", "-u", "origin", "main")
	return clone
}

func TestDispatchOpensNotePR(t *testing.T) {
	repo := newRepoForRunner(t)
	orch, err := gitops.New(gitops.Options{
		RepoRoot:   repo,
		BaseBranch: "main",
		SkipPR:     true, // don't hit gh in tests
	})
	if err != nil {
		t.Fatalf("gitops.New: %v", err)
	}

	fam := family.NewStore()
	defaults, _ := family.LoadDefaults()
	fam.Seed(defaults)
	db, err := storage.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	r, err := New(Deps{
		Family:   fam,
		Duties:   duty.NewBuiltinRegistry(),
		Storage:  db,
		Provider: provider.NewMock(),
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		GitOps:   orch,
		RepoRoot: repo,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	run, err := r.Dispatch(context.Background(), DispatchRequest{Member: "jerry", Duty: "lint-fix"})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if run.Status != storage.RunSucceeded {
		t.Fatalf("status = %q", run.Status)
	}
	if run.Branch == nil || !strings.HasPrefix(*run.Branch, "night-family/jerry/lint-fix-") {
		t.Errorf("branch = %v", run.Branch)
	}
	// PR URL is empty because we SkipPR'd. That's the contract for
	// this test.
	if run.PRURL != nil && *run.PRURL != "" {
		t.Errorf("unexpected PR URL (tests run with SkipPR): %q", *run.PRURL)
	}
}
