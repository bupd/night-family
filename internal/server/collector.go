package server

import (
	"fmt"
	"io"
	"sort"
	"sync"
)

// httpCollector tracks per-route HTTP request counts and cumulative
// durations. It is goroutine-safe and intended to be populated by the
// logging middleware and read by the /metrics handler.
type httpCollector struct {
	mu       sync.Mutex
	counts   map[routeKey]int64
	durSumMs map[durationKey]float64 // cumulative seconds
}

type routeKey struct {
	method string
	route  string
	status int
}

type durationKey struct {
	method string
	route  string
}

func newHTTPCollector() *httpCollector {
	return &httpCollector{
		counts:   make(map[routeKey]int64),
		durSumMs: make(map[durationKey]float64),
	}
}

// record is called once per completed request.
func (c *httpCollector) record(method, route string, status int, durationSec float64) {
	c.mu.Lock()
	c.counts[routeKey{method, route, status}]++
	c.durSumMs[durationKey{method, route}] += durationSec
	c.mu.Unlock()
}

// writePrometheus writes nf_http_requests_total and
// nf_http_request_duration_seconds metrics in Prometheus text format.
func (c *httpCollector) writePrometheus(w io.Writer) {
	c.mu.Lock()
	// snapshot under lock
	counts := make([]struct {
		k routeKey
		v int64
	}, 0, len(c.counts))
	for k, v := range c.counts {
		counts = append(counts, struct {
			k routeKey
			v int64
		}{k, v})
	}
	durations := make([]struct {
		k durationKey
		v float64
	}, 0, len(c.durSumMs))
	for k, v := range c.durSumMs {
		durations = append(durations, struct {
			k durationKey
			v float64
		}{k, v})
	}
	c.mu.Unlock()

	if len(counts) > 0 {
		sort.Slice(counts, func(i, j int) bool {
			a, b := counts[i].k, counts[j].k
			if a.method != b.method {
				return a.method < b.method
			}
			if a.route != b.route {
				return a.route < b.route
			}
			return a.status < b.status
		})
		fmt.Fprintf(w, "# HELP nf_http_requests_total Total HTTP requests by method, route, and status.\n")
		fmt.Fprintf(w, "# TYPE nf_http_requests_total counter\n")
		for _, e := range counts {
			fmt.Fprintf(w, "nf_http_requests_total{method=%q,route=%q,status=\"%d\"} %d\n",
				e.k.method, e.k.route, e.k.status, e.v)
		}
	}

	if len(durations) > 0 {
		sort.Slice(durations, func(i, j int) bool {
			a, b := durations[i].k, durations[j].k
			if a.method != b.method {
				return a.method < b.method
			}
			return a.route < b.route
		})
		fmt.Fprintf(w, "# HELP nf_http_request_duration_seconds Cumulative HTTP request duration in seconds.\n")
		fmt.Fprintf(w, "# TYPE nf_http_request_duration_seconds summary\n")
		for _, e := range durations {
			var total int64
			for _, c := range counts {
				if c.k.method == e.k.method && c.k.route == e.k.route {
					total += c.v
				}
			}
			fmt.Fprintf(w, "nf_http_request_duration_seconds_count{method=%q,route=%q} %d\n",
				e.k.method, e.k.route, total)
			fmt.Fprintf(w, "nf_http_request_duration_seconds_sum{method=%q,route=%q} %f\n",
				e.k.method, e.k.route, e.v)
		}
	}
}
