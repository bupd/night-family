package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// bootedAt is stamped when the server struct is constructed (via its
// package-level init) so /metrics can report process uptime without
// each Server instance tracking it.
var bootedAt = time.Now()

func (s *Server) metricsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /metrics", s.metrics)
}

// metrics emits a minimal hand-rolled Prometheus text-format payload.
// We avoid pulling in prometheus/client_golang because the metric set
// is small and rigidly shaped — a few counters derived from
// storage.Stats + an uptime gauge. If the metric surface grows past
// ~dozen series we should revisit.
func (s *Server) metrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	fmt.Fprintf(w, "# HELP nf_up Whether the daemon is serving requests.\n")
	fmt.Fprintf(w, "# TYPE nf_up gauge\n")
	fmt.Fprintf(w, "nf_up 1\n")

	fmt.Fprintf(w, "# HELP nf_uptime_seconds Seconds since the daemon booted.\n")
	fmt.Fprintf(w, "# TYPE nf_uptime_seconds counter\n")
	fmt.Fprintf(w, "nf_uptime_seconds %f\n", time.Since(bootedAt).Seconds())

	if s.cfg.Storage == nil {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	stats, err := s.cfg.Storage.Stats(ctx)
	if err != nil {
		_, _ = io.WriteString(w, "# stats: error\n")
		return
	}

	fmt.Fprintf(w, "# HELP nf_nights_total Number of nights on record.\n")
	fmt.Fprintf(w, "# TYPE nf_nights_total counter\n")
	fmt.Fprintf(w, "nf_nights_total %d\n", stats.Nights)

	fmt.Fprintf(w, "# HELP nf_runs_total Number of dispatched runs.\n")
	fmt.Fprintf(w, "# TYPE nf_runs_total counter\n")
	fmt.Fprintf(w, "nf_runs_total %d\n", stats.Runs)

	fmt.Fprintf(w, "# HELP nf_runs_by_status Runs bucketed by terminal status.\n")
	fmt.Fprintf(w, "# TYPE nf_runs_by_status counter\n")
	for status, n := range stats.RunsByStatus {
		fmt.Fprintf(w, "nf_runs_by_status{status=%q} %d\n", status, n)
	}

	fmt.Fprintf(w, "# HELP nf_prs_total Number of PRs opened.\n")
	fmt.Fprintf(w, "# TYPE nf_prs_total counter\n")
	fmt.Fprintf(w, "nf_prs_total %d\n", stats.PRs)

	fmt.Fprintf(w, "# HELP nf_prs_by_state PRs bucketed by GitHub state.\n")
	fmt.Fprintf(w, "# TYPE nf_prs_by_state counter\n")
	for state, n := range stats.PRsByState {
		fmt.Fprintf(w, "nf_prs_by_state{state=%q} %d\n", state, n)
	}
}
