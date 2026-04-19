package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Claude invokes `claude` in print mode (`claude --print`) with a prompt
// synthesised from the family member's system prompt and the duty
// description. Output is captured into Result.Summary.
//
// This is a deliberately thin adapter: we don't parse a structured
// response, we don't pipe ongoing file edits back in real time. If
// the model writes files directly (which the claude CLI does when its
// permission mode is permissive), those edits live in the repo
// checkout when the subprocess exits — gitops picks them up from
// there. Runner's note-PR fallback still fires if no files changed.
type Claude struct {
	// Bin is the claude binary path. Default: "claude".
	Bin string
	// ExtraArgs is appended after the defaults. Handy for things like
	// "--dangerously-skip-permissions" or "--model claude-opus-4-7".
	ExtraArgs []string
	// Timeout caps how long a single Run blocks. Default: 20 minutes.
	Timeout time.Duration
}

// NewClaude returns a Claude provider with sensible defaults.
func NewClaude() *Claude {
	return &Claude{
		Bin:     "claude",
		Timeout: 20 * time.Minute,
	}
}

// Name is "claude".
func (c *Claude) Name() string { return "claude" }

// Run invokes `claude --print` with a combined prompt. stdout becomes
// Summary; stderr is folded into the error on non-zero exit.
func (c *Claude) Run(ctx context.Context, req Request) (*Result, error) {
	if c.Bin == "" {
		c.Bin = "claude"
	}
	if c.Timeout == 0 {
		c.Timeout = 20 * time.Minute
	}
	if _, err := exec.LookPath(c.Bin); err != nil {
		return &Result{Err: fmt.Errorf("claude: binary %q not found on $PATH: %w", c.Bin, err)},
			fmt.Errorf("claude binary not available")
	}

	prompt := composePrompt(req)

	// Subprocess-local ctx: we want to respect the caller's ctx but also
	// enforce our own Timeout ceiling.
	cctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	args := append([]string{"--print"}, c.ExtraArgs...)
	cmd := exec.CommandContext(cctx, c.Bin, args...)
	if req.RepoRoot != "" {
		cmd.Dir = req.RepoRoot
	}
	cmd.Stdin = strings.NewReader(prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	out := stdout.String()
	if err != nil {
		if errors.Is(cctx.Err(), context.DeadlineExceeded) {
			return &Result{Err: fmt.Errorf("claude: timed out after %s", c.Timeout)}, nil
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return &Result{Err: ctx.Err()}, ctx.Err()
		}
		return &Result{
			Err:     fmt.Errorf("claude exited with error: %w\n%s", err, strings.TrimSpace(stderr.String())),
			Summary: out,
		}, nil
	}
	return &Result{
		Summary: strings.TrimSpace(out),
	}, nil
}

// composePrompt builds the input we hand to claude --print on stdin.
// Keeps the member's persona + the duty's description + any duty-
// specific context visible to the model in one pass.
func composePrompt(req Request) string {
	var b strings.Builder
	if req.MemberPrompt != "" {
		b.WriteString("# Persona\n\n")
		b.WriteString(strings.TrimSpace(req.MemberPrompt))
		b.WriteString("\n\n")
	}
	b.WriteString("# Task\n\n")
	b.WriteString("You are ")
	b.WriteString(req.Member)
	b.WriteString(". Tonight's duty: ")
	b.WriteString(req.Duty)
	b.WriteString(".\n\n")
	if req.DutyPrompt != "" {
		b.WriteString(strings.TrimSpace(req.DutyPrompt))
		b.WriteString("\n\n")
	}
	b.WriteString("# Repository\n\n")
	b.WriteString("Working directory: ")
	b.WriteString(req.RepoRoot)
	b.WriteString("\n\n")
	b.WriteString("# Output expectations\n\n")
	b.WriteString("- Keep any edits small and scoped to this duty.\n")
	b.WriteString("- When you finish, print a short markdown summary to stdout — that text becomes the PR body.\n")
	b.WriteString("- Never commit or push; night-family handles git for you.\n")
	b.WriteString("- Never modify files outside the working directory.\n")
	if len(req.Args) > 0 {
		b.WriteString("\n# Duty args\n\n")
		for k, v := range req.Args {
			fmt.Fprintf(&b, "- %s: %v\n", k, v)
		}
	}
	return b.String()
}
