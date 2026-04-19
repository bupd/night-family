package gitops

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newRepo creates a bare git repo on disk + a clone of it, so push
// succeeds (it has somewhere to push to) without needing the network.
// Returns the clone path.
func newRepo(t *testing.T) string {
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

func TestOpenFullLocalFlow(t *testing.T) {
	repo := newRepo(t)
	o, err := New(Options{
		RepoRoot:     repo,
		BaseBranch:   "main",
		BranchPrefix: "night-family/",
		SkipPR:       true, // no gh in test
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	res, err := o.Open(context.Background(), OpenRequest{
		Branch:    "note/smoke",
		CommitMsg: "note: smoke test",
		PRTitle:   "note: smoke",
		PRBody:    "body",
		Changes: []Change{
			{Path: ".night-family/notes/smoke.md", Content: "hello from the family\n"},
		},
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if res.Branch != "night-family/note/smoke" {
		t.Errorf("branch = %q", res.Branch)
	}
	if !res.Pushed {
		t.Errorf("not pushed")
	}
	if res.Skipped != "pr" {
		t.Errorf("Skipped = %q, want pr", res.Skipped)
	}

	// The file should exist on the new branch in the clone.
	data, err := os.ReadFile(filepath.Join(repo, ".night-family/notes/smoke.md"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !strings.Contains(string(data), "hello from the family") {
		t.Errorf("file content unexpected: %q", string(data))
	}

	// origin should now have the branch too (push happened).
	originDir := filepath.Dir(repo) + "/origin.git"
	cmd := exec.Command("git", "--git-dir", originDir, "branch")
	out, _ := cmd.CombinedOutput()
	if !strings.Contains(string(out), "night-family/note/smoke") {
		t.Errorf("origin missing pushed branch:\n%s", out)
	}
}

func TestOpenSkipPushLeavesBranchLocal(t *testing.T) {
	repo := newRepo(t)
	o, _ := New(Options{
		RepoRoot: repo, BaseBranch: "main",
		SkipPush: true, SkipPR: true,
	})
	res, err := o.Open(context.Background(), OpenRequest{
		Branch:    "note/local",
		CommitMsg: "note: local only",
		Changes:   []Change{{Path: "a.txt", Content: "x"}},
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if res.Pushed {
		t.Errorf("push should have been skipped")
	}
	if res.Skipped != "push" {
		t.Errorf("Skipped = %q, want push", res.Skipped)
	}
	if res.Commit == "" {
		t.Errorf("no commit sha")
	}
}

func TestOpenRejectsMissingBranch(t *testing.T) {
	repo := newRepo(t)
	o, _ := New(Options{RepoRoot: repo, SkipPush: true, SkipPR: true})
	_, err := o.Open(context.Background(), OpenRequest{CommitMsg: "x"})
	if err == nil {
		t.Fatalf("expected error for empty Branch")
	}
}
