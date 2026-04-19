package server

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/bupd/night-family/internal/storage"
)

func (s *Server) runsRoutes(mux *http.ServeMux) {
	if s.cfg.Storage == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/runs", s.listRuns)
	mux.HandleFunc("GET /api/v1/runs/{id}", s.getRun)
	mux.HandleFunc("GET /runs", s.runsPage)
}

func (s *Server) listRuns(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	f := storage.ListRunsFilter{
		Member: r.URL.Query().Get("member"),
		Duty:   r.URL.Query().Get("duty"),
		Status: storage.RunStatus(r.URL.Query().Get("status")),
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			f.Limit = n
		}
	}
	runs, err := s.cfg.Storage.ListRuns(ctx, f)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "db_error", err.Error(), r.URL.Path)
		return
	}
	if runs == nil {
		runs = []storage.Run{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": runs})
}

func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	id := r.PathValue("id")
	run, err := s.cfg.Storage.GetRun(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		writeProblem(w, http.StatusNotFound, "not_found", "Run not found", r.URL.Path)
		return
	}
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "db_error", err.Error(), r.URL.Path)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) runsPage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	runs, err := s.cfg.Storage.ListRuns(ctx, storage.ListRunsFilter{Limit: 100})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct {
		Title   string
		Version string
		Runs    []storage.Run
	}{
		Title: "Runs — night-family",
		Runs:  runs,
	}
	s.renderPage(w, "runs", data)
}
