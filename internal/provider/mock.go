package provider

import (
	"context"
	"fmt"
	"time"
)

// Mock is a zero-side-effect provider used in tests, local
// development, and the v1 daemon before a real adapter is wired.
// Every Run call returns a canned Result after a configurable delay,
// making it easy to exercise the full runner → storage → API path
// without hitting the network or spawning a subprocess.
type Mock struct {
	// Delay is how long Run pretends to work for.
	Delay time.Duration
	// TokensIn / TokensOut are what Run reports as consumed.
	TokensIn, TokensOut int
	// FailMember, when non-empty, causes Run to return an error for
	// requests coming from that member — lets tests exercise the
	// failed-run path.
	FailMember string
}

// NewMock returns a Mock with sensible defaults.
func NewMock() *Mock {
	return &Mock{
		Delay:     20 * time.Millisecond,
		TokensIn:  1234,
		TokensOut: 567,
	}
}

// Name is "mock".
func (m *Mock) Name() string { return "mock" }

// Run implements Provider. Honours ctx cancellation.
func (m *Mock) Run(ctx context.Context, req Request) (*Result, error) {
	if m.FailMember != "" && req.Member == m.FailMember {
		return &Result{Err: fmt.Errorf("mock: forced failure for member %q", req.Member)}, nil
	}
	select {
	case <-time.After(m.Delay):
	case <-ctx.Done():
		return &Result{Err: ctx.Err()}, ctx.Err()
	}
	return &Result{
		Summary: fmt.Sprintf(
			"[mock] %s/%s completed — no real changes made. "+
				"This is a simulated run so the rest of the pipeline can "+
				"be tested end-to-end without burning provider tokens.",
			req.Member, req.Duty),
		TokensIn:  m.TokensIn,
		TokensOut: m.TokensOut,
	}, nil
}
