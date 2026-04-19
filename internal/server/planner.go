package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/bupd/night-family/internal/planner"
)

func (s *Server) plannerRoutes(mux *http.ServeMux) {
	if s.cfg.Family == nil || s.cfg.Duties == nil || s.cfg.Schedule == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/nights/preview", s.previewNight)
	mux.HandleFunc("GET /plan", s.planPage)
}

func (s *Server) buildPlanFromRequest(r *http.Request) (planner.Plan, error) {
	now := time.Now()
	if s.cfg.Clock != nil {
		now = s.cfg.Clock()
	}
	budget := 0
	if b := r.URL.Query().Get("budget"); b != "" {
		if n, err := strconv.Atoi(b); err == nil && n > 0 {
			budget = n
		}
	}
	return planner.Input{
		Family:       s.cfg.Family,
		Duties:       s.cfg.Duties,
		Schedule:     s.cfg.Schedule,
		Now:          now,
		BudgetTokens: budget,
	}.Plan()
}

func (s *Server) previewNight(w http.ResponseWriter, r *http.Request) {
	plan, err := s.buildPlanFromRequest(r)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "plan_failed", err.Error(), r.URL.Path)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (s *Server) planPage(w http.ResponseWriter, r *http.Request) {
	plan, err := s.buildPlanFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct {
		Title   string
		Version string
		Plan    planner.Plan
	}{
		Title: "Tonight's plan — night-family",
		Plan:  plan,
	}
	s.renderPage(w, "plan", data)
}
