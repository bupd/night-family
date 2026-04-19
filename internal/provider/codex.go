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

// Codex invokes the `codex` CLI (OpenAI's Codex command-line agent)
// in a non-interactive way. The shape is intentionally close to
// Claude — one shell-out per run, prompt on stdin, summary on stdout,
// stderr folded into the error on non-zero exit.
//
// If the codex CLI later grows a different non-interactive flag
// surface, the adapter is small enough to update in one place.
type Codex struct {
	// Bin is the codex binary path. Default: "codex".
	Bin string
	// ExtraArgs is appended after the defaults.
	ExtraArgs []string
	// Timeout caps how long a single Run blocks. Default: 20 minutes.
	Timeout time.Duration
}

// NewCodex returns a Codex provider with sensible defaults.
func NewCodex() *Codex {
	return &Codex{
		Bin:     "codex",
		Timeout: 20 * time.Minute,
	}
}

// Name is "codex".
func (c *Codex) Name() string { return "codex" }

// Run spawns the codex CLI with the common night-family prompt.
func (c *Codex) Run(ctx context.Context, req Request) (*Result, error) {
	if c.Bin == "" {
		c.Bin = "codex"
	}
	if c.Timeout == 0 {
		c.Timeout = 20 * time.Minute
	}
	if _, err := exec.LookPath(c.Bin); err != nil {
		return &Result{Err: fmt.Errorf("codex: binary %q not found on $PATH: %w", c.Bin, err)},
			fmt.Errorf("codex binary not available")
	}

	prompt := composePrompt(req)

	cctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	// v1: non-interactive single-shot via stdin. If the codex CLI
	// grows a specific flag for this we'll switch to it.
	args := append([]string{}, c.ExtraArgs...)
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
			return &Result{Err: fmt.Errorf("codex: timed out after %s", c.Timeout)}, nil
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return &Result{Err: ctx.Err()}, ctx.Err()
		}
		return &Result{
			Err:     fmt.Errorf("codex exited with error: %w\n%s", err, strings.TrimSpace(stderr.String())),
			Summary: out,
		}, nil
	}
	return &Result{Summary: strings.TrimSpace(out)}, nil
}
