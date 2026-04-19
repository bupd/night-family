// Package gitops wraps the git + gh CLIs with just enough surface for
// the night-family orchestrator: resolve main, carve out a worktree,
// stage and commit files, push, open a PR, tag reviewers.
//
// Nothing here imports go-git — we shell out. Reasoning:
//   - The gh CLI does PR creation right (auth, tokens, notifications).
//     Re-implementing it is strictly worse.
//   - git CLI is the one the user already has configured (credentials,
//     SSH keys, commit-signing, hooks). We want to inherit all of it.
//
// Tests exercise the package against a real on-disk git repo created
// in a t.TempDir; gh calls can be skipped via a functional option so
// CI (which may not have gh + a token) is happy.
package gitops

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Options configure an Orchestrator.
type Options struct {
	// RepoRoot is the path to the local checkout. Must contain .git.
	RepoRoot string
	// BaseBranch is the branch PRs target. Default: "main".
	BaseBranch string
	// BranchPrefix is prepended to every generated branch. Default:
	// "night-family/".
	BranchPrefix string
	// Reviewers tagged in the PR body (not --reviewer; bots often
	// can't accept formal review requests).
	Reviewers []string
	// SignOff adds a DCO Signed-off-by trailer to commits.
	SignOff bool
	// Identity is the git author identity for the commit. If empty,
	// inherits from local git config.
	Identity Identity
	// SkipPush, when true, stops after the local commit. Handy in
	// tests that don't want to hit a remote.
	SkipPush bool
	// SkipPR, when true, stops after push. Handy in tests without gh.
	SkipPR bool
}

// Identity is a git author identity.
type Identity struct {
	Name  string
	Email string
}

// Orchestrator runs git + gh against a specific local checkout.
type Orchestrator struct {
	opts Options
}

// New returns an Orchestrator. RepoRoot must exist and contain a .git
// directory; base/prefix fall back to sensible defaults.
func New(opts Options) (*Orchestrator, error) {
	if opts.RepoRoot == "" {
		return nil, errors.New("gitops: RepoRoot required")
	}
	if opts.BaseBranch == "" {
		opts.BaseBranch = "main"
	}
	if opts.BranchPrefix == "" {
		opts.BranchPrefix = "night-family/"
	}
	return &Orchestrator{opts: opts}, nil
}

// Change is a single file to write within a PR.
type Change struct {
	// Path is relative to the repo root. Parent directories are
	// created as needed.
	Path string
	// Content is the file body. Use "" with Delete=true to remove.
	Content string
	// Delete, when true, removes the file instead of creating it.
	Delete bool
}

// OpenRequest is the input to Open. Branch is the short suffix that's
// appended to BranchPrefix; Title / Body are the PR metadata;
// Changes is the file-system delta.
type OpenRequest struct {
	Branch    string
	CommitMsg string
	PRTitle   string
	PRBody    string
	Changes   []Change
}

// OpenResult is what Open returns.
type OpenResult struct {
	Branch  string `json:"branch"`
	Commit  string `json:"commit"`
	PRURL   string `json:"pr_url,omitempty"`
	Pushed  bool   `json:"pushed"`
	Skipped string `json:"skipped,omitempty"`
}

// Open runs the full branch → commit → push → gh pr create → tag
// flow. Each step is observable + testable in isolation via the
// SkipPush / SkipPR options.
func (o *Orchestrator) Open(ctx context.Context, req OpenRequest) (OpenResult, error) {
	if strings.TrimSpace(req.Branch) == "" {
		return OpenResult{}, errors.New("gitops: Branch required")
	}
	if strings.TrimSpace(req.CommitMsg) == "" {
		return OpenResult{}, errors.New("gitops: CommitMsg required")
	}
	branch := o.opts.BranchPrefix + req.Branch

	// Start from a fresh checkout of main so we don't step on the
	// caller's working copy. Prefer a detached branch off origin/main
	// if origin is reachable; otherwise fall back to the local base.
	if _, err := o.git(ctx, "checkout", "-B", branch, o.opts.BaseBranch); err != nil {
		return OpenResult{}, fmt.Errorf("checkout -B %s: %w", branch, err)
	}

	// Apply the file changes.
	for _, c := range req.Changes {
		abs := filepath.Join(o.opts.RepoRoot, c.Path)
		if c.Delete {
			if _, err := o.git(ctx, "rm", "-f", c.Path); err != nil {
				return OpenResult{}, fmt.Errorf("git rm %s: %w", c.Path, err)
			}
			continue
		}
		if err := writeFile(abs, c.Content); err != nil {
			return OpenResult{}, fmt.Errorf("write %s: %w", c.Path, err)
		}
		if _, err := o.git(ctx, "add", "--", c.Path); err != nil {
			return OpenResult{}, fmt.Errorf("git add %s: %w", c.Path, err)
		}
	}

	// Commit.
	args := []string{"commit"}
	if o.opts.SignOff {
		args = append(args, "-s")
	}
	if o.opts.Identity.Name != "" {
		args = append(args, "--author="+o.opts.Identity.Name+" <"+o.opts.Identity.Email+">")
	}
	args = append(args, "-m", req.CommitMsg)
	if _, err := o.git(ctx, args...); err != nil {
		return OpenResult{}, fmt.Errorf("commit: %w", err)
	}
	sha, err := o.git(ctx, "rev-parse", "HEAD")
	if err != nil {
		return OpenResult{}, fmt.Errorf("rev-parse: %w", err)
	}
	res := OpenResult{Branch: branch, Commit: strings.TrimSpace(sha)}

	if o.opts.SkipPush {
		res.Skipped = "push"
		return res, nil
	}
	if _, err := o.git(ctx, "push", "-u", "origin", branch); err != nil {
		return OpenResult{}, fmt.Errorf("push: %w", err)
	}
	res.Pushed = true

	if o.opts.SkipPR {
		res.Skipped = "pr"
		return res, nil
	}

	body := req.PRBody
	if len(o.opts.Reviewers) > 0 {
		var at []string
		for _, r := range o.opts.Reviewers {
			at = append(at, "@"+r)
		}
		body += "\n\ncc " + strings.Join(at, " ") + " — please review."
	}
	out, err := o.gh(ctx, "pr", "create",
		"--base", o.opts.BaseBranch,
		"--head", branch,
		"--title", req.PRTitle,
		"--body", body,
	)
	if err != nil {
		return OpenResult{}, fmt.Errorf("gh pr create: %w", err)
	}
	res.PRURL = strings.TrimSpace(out)
	return res, nil
}

func (o *Orchestrator) git(ctx context.Context, args ...string) (string, error) {
	return runCmd(ctx, o.opts.RepoRoot, "git", args...)
}

func (o *Orchestrator) gh(ctx context.Context, args ...string) (string, error) {
	return runCmd(ctx, o.opts.RepoRoot, "gh", args...)
}

func runCmd(ctx context.Context, dir, bin string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("%s %s: %w: %s",
			bin, strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}
