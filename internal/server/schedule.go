package server

import (
	"net/http"
	"time"
)

func (s *Server) scheduleRoutes(mux *http.ServeMux) {
	if s.cfg.Schedule == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/schedule", s.getSchedule)
}

func (s *Server) getSchedule(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	if s.cfg.Clock != nil {
		now = s.cfg.Clock()
	}
	sum, err := s.cfg.Schedule.Summarize(now)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "invalid_schedule", err.Error(), r.URL.Path)
		return
	}
	writeJSON(w, http.StatusOK, sum)
}
