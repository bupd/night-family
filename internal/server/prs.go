package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/bupd/night-family/internal/storage"
)

func (s *Server) prsRoutes(mux *http.ServeMux) {
	if s.cfg.Storage == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/prs", s.listPRs)
	mux.HandleFunc("GET /api/v1/prs/{id}", s.getPR)
	mux.HandleFunc("GET /prs", s.prsPage)
}

func (s *Server) listPRs(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	prs, err := s.cfg.Storage.ListPRs(ctx, 100)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "db_error", err.Error(), r.URL.Path)
		return
	}
	if prs == nil {
		prs = []storage.PR{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": prs})
}

func (s *Server) getPR(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	id := r.PathValue("id")
	pr, err := s.cfg.Storage.GetPR(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		writeProblem(w, http.StatusNotFound, "not_found", "PR not found", r.URL.Path)
		return
	}
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "db_error", err.Error(), r.URL.Path)
		return
	}
	writeJSON(w, http.StatusOK, pr)
}

func (s *Server) prsPage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	prs, err := s.cfg.Storage.ListPRs(ctx, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct {
		Title   string
		Version string
		PRs     []storage.PR
	}{
		Title: "PRs — night-family",
		PRs:   prs,
	}
	s.renderPage(w, "prs", data)
}
