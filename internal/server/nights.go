package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/bupd/night-family/internal/runner"
	"github.com/bupd/night-family/internal/storage"
)

func (s *Server) nightsRoutes(mux *http.ServeMux) {
	if s.cfg.Storage == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/nights", s.listNights)
	mux.HandleFunc("GET /api/v1/nights/{id}", s.getNight)
	if s.cfg.Runner != nil && s.cfg.Schedule != nil {
		mux.HandleFunc("POST /api/v1/nights/trigger", s.triggerNight)
	}
}

func (s *Server) listNights(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	nights, err := s.cfg.Storage.ListNights(ctx, 100)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "db_error", err.Error(), r.URL.Path)
		return
	}
	if nights == nil {
		nights = []storage.Night{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": nights})
}

func (s *Server) getNight(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, night)
}

type triggerNightRequest struct {
	OnlyMembers []string `json:"only_members,omitempty"`
	OnlyDuties  []string `json:"only_duties,omitempty"`
	Budget      int      `json:"budget,omitempty"`
	DryRun      bool     `json:"dry_run,omitempty"`
}

func (s *Server) triggerNight(w http.ResponseWriter, r *http.Request) {
	var req triggerNightRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeProblem(w, http.StatusBadRequest, "bad_request", "invalid JSON body: "+err.Error(), r.URL.Path)
			return
		}
	}
	// Give the night a generous ceiling; individual mock runs are
	// ~20ms each, so a full plan finishes well inside this.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	res, err := s.cfg.Runner.TriggerNight(ctx, s.cfg.Schedule, runner.NightOptions{
		OnlyMembers: req.OnlyMembers,
		OnlyDuties:  req.OnlyDuties,
		Budget:      req.Budget,
		DryRun:      req.DryRun,
	})
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "trigger_failed", err.Error(), r.URL.Path)
		return
	}
	writeJSON(w, http.StatusAccepted, res)
}
