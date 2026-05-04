package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/bupd/night-family/internal/digest"
	"github.com/bupd/night-family/internal/storage"
)

func (s *Server) digestPageRoutes(mux *http.ServeMux) {
	if s.cfg.Storage == nil {
		return
	}
	mux.HandleFunc("GET /digests", s.digestsPage)
	mux.HandleFunc("GET /digests/{id}", s.digestPage)
}

func (s *Server) digestsPage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	nights, err := s.cfg.Storage.ListNights(ctx, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct {
		Title   string
		Version string
		Nights  []storage.Night
	}{
		Title:  "Digests — night-family",
		Nights: nights,
	}
	s.renderPage(w, "digests", data)
}

func (s *Server) digestPage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	id := r.PathValue("id")
	night, err := s.cfg.Storage.GetNight(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	runs, err := s.cfg.Storage.ListRuns(ctx, storage.ListRunsFilter{Limit: 500})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	filtered := runs[:0]
	for _, rn := range runs {
		if rn.NightID != nil && *rn.NightID == id {
			filtered = append(filtered, rn)
		}
	}
	prs, err := s.cfg.Storage.ListPRs(ctx, 500)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
	data := struct {
		Title   string
		Version string
		NightID string
		Body    string
	}{
		Title:   "Digest — " + id,
		NightID: id,
		Body:    body,
	}
	s.renderPage(w, "digest", data)
}
