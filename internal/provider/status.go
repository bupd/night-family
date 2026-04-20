package provider

import (
	"context"
	"time"
)

// SessionStatus lets the Mock provider surface a plausible status so
// the dashboard has something to show during demos + tests. Tokens
// tick down as Mock is used so consumers can observe motion.
func (m *Mock) SessionStatus(_ context.Context) (SessionStatus, error) {
	return SessionStatus{
		Provider:                m.Name(),
		RemainingTokensEstimate: 100_000,
		WindowEndsAt:            time.Now().Add(5 * time.Hour),
		Confidence:              "low",
	}, nil
}

// SessionStatus for Claude is a stub until we pick a probe strategy.
// Returns "unknown" with Confidence=low so callers render a sensible
// placeholder rather than crashing. Real parsing lands in a follow-up.
func (c *Claude) SessionStatus(_ context.Context) (SessionStatus, error) {
	return SessionStatus{
		Provider:   c.Name(),
		Confidence: "low",
	}, nil
}

// SessionStatus for Codex mirrors Claude's stub for now.
func (c *Codex) SessionStatus(_ context.Context) (SessionStatus, error) {
	return SessionStatus{
		Provider:   c.Name(),
		Confidence: "low",
	}, nil
}
