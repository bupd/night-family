package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/bupd/night-family/internal/digest"
	"github.com/bupd/night-family/internal/storage"
)

func (s *Server) digestRoutes(mux *http.ServeMux) {
	if s.cfg.Storage == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/nights/{id}/digest", s.getNightDigest)
}

// getNightDigest renders the digest for a night on demand, pulled from
// live DB state so clients don't have to rely on the on-disk file the
// runner writes at FinishNight. Served as text/markdown so piping
// `curl … > out.md` Just Works.
func (s *Server) getNightDigest(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	id := r.PathValue("id")
	night, err := s.cfg.Storage.GetNight(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		writeProblem(w, http.StatusNotFound, "not_found", "Night not found", r.URL.Path)
		return
	}
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "db_error", err.Error(), r.URL.Path)
		return
	}

	runs, err := s.cfg.Storage.ListRuns(ctx, storage.ListRunsFilter{Limit: 500})
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "db_error", err.Error(), r.URL.Path)
		return
	}
	// Filter to runs whose night_id matches.
	filtered := runs[:0]
	for _, rn := range runs {
		if rn.NightID != nil && *rn.NightID == id {
			filtered = append(filtered, rn)
		}
	}

	prs, _ := s.cfg.Storage.ListPRs(ctx, 500)
	byRun := map[string]bool{}
	for _, rn := range filtered {
		byRun[rn.ID] = true
	}
	filteredPRs := prs[:0]
	for _, p := range prs {
		if p.RunID != nil && byRun[*p.RunID] {
			filteredPRs = append(filteredPRs, p)
		}
	}

	body := digest.Render(digest.Night{Night: night, Runs: filtered, PRs: filteredPRs})

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	_, _ = w.Write([]byte(body))
}
