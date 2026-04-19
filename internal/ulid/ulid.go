// Package ulid generates prefixed ULIDs for night-family's resource
// IDs (run_…, night_…, pr_…). ULIDs are sortable by time, which lines
// up with our DB indexes.
package ulid

import (
	"math/rand"
	"sync"
	"time"

	gulid "github.com/oklog/ulid/v2"
)

// generator wraps the monotonic reader so concurrent Make calls don't
// race on the underlying source.
type generator struct {
	mu     sync.Mutex
	reader *gulid.MonotonicEntropy
}

var defaultGen = newGenerator(time.Now)

func newGenerator(_ func() time.Time) *generator {
	seed := time.Now().UnixNano()
	//nolint:gosec // Non-crypto randomness is fine for an opaque ID tiebreaker.
	r := rand.New(rand.NewSource(seed))
	return &generator{reader: gulid.Monotonic(r, 0)}
}

// Make returns "<prefix>_<26-char ULID>" for the given kind, e.g.
// Make("run") → "run_01HTEST…".
func Make(prefix string) string {
	return defaultGen.make(prefix, time.Now())
}

// MakeAt is Make with an explicit timestamp, for tests.
func MakeAt(prefix string, t time.Time) string {
	return defaultGen.make(prefix, t)
}

func (g *generator) make(prefix string, t time.Time) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	id := gulid.MustNew(gulid.Timestamp(t), g.reader)
	return prefix + "_" + id.String()
}
