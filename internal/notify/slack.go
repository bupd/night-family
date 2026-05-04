// Package notify ships night summaries to external sinks. v1 ships
// one: a Slack-compatible incoming webhook. Discord + generic
// webhooks work too — they all accept the same `{"text": "…"}` shape.
//
// The Notifier interface keeps the wiring open for email, SMS, or
// desktop notifications later without touching the runner.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Notifier is the seam the runner calls once per finished night.
type Notifier interface {
	Notify(ctx context.Context, title, body string) error
}

// Noop is a safe default — when no webhook is configured, use this
// so callers don't have to nil-check.
type Noop struct{}

func (Noop) Notify(_ context.Context, _, _ string) error { return nil }

// Slack posts JSON with a `text` field to an incoming-webhook URL.
// Compatible with Slack, Discord (with `/slack` suffix on the
// webhook), and anything that accepts `{"text": "..."}`.
type Slack struct {
	WebhookURL string
	// Timeout caps a single post. Default: 10s.
	Timeout time.Duration
	// Client overrides http.DefaultClient. Tests use httptest.Server
	// via this hook.
	Client *http.Client
}

// NewSlack constructs a Slack notifier with sensible defaults.
func NewSlack(url string) *Slack {
	return &Slack{WebhookURL: url, Timeout: 10 * time.Second}
}

// Notify posts "<title>\n\n<body>" (body truncated at 3000 chars).
func (s *Slack) Notify(ctx context.Context, title, body string) error {
	if s.WebhookURL == "" {
		return nil
	}
	if s.Timeout == 0 {
		s.Timeout = 10 * time.Second
	}
	if len(body) > 3000 {
		body = body[:3000] + "\n…(truncated)"
	}
	payload := map[string]string{"text": title + "\n\n" + body}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("slack: marshal payload: %w", err)
	}

	cctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodPost, s.WebhookURL, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	c := s.Client
	if c == nil {
		c = http.DefaultClient
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return fmt.Errorf("slack webhook returned %s: %s", resp.Status, string(b))
	}
	return nil
}
