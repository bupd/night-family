package server

import (
	"context"
	"net/http"
	"time"

	"github.com/bupd/night-family/internal/schedule"
	"github.com/bupd/night-family/internal/storage"
)

func (s *Server) dashboardRoutes(mux *http.ServeMux) {
	if s.cfg.Storage == nil {
		return
	}
	mux.HandleFunc("GET /ui/dashboard-cards", s.dashboardCards)
}

type dashboardData struct {
	Stats     storage.Stats
	Schedule  schedule.Schedule
	InWindow  bool
	NextStart time.Time
}

func (s *Server) dashboardCards(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	st, err := s.cfg.Storage.Stats(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := dashboardData{Stats: st}
	if s.cfg.Schedule != nil {
		data.Schedule = *s.cfg.Schedule
		now := time.Now()
		if s.cfg.Clock != nil {
			now = s.cfg.Clock()
		}
		start, _, err := s.cfg.Schedule.Next(now)
		if err == nil {
			data.NextStart = start
		}
		data.InWindow = s.cfg.Schedule.IsInWindow(now)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tpl := s.dashCardTpl
	if tpl == nil {
		http.Error(w, "dashboard template not parsed", http.StatusInternalServerError)
		return
	}
	if err := tpl.ExecuteTemplate(w, "_dashboard_cards", data); err != nil {
		s.cfg.Logger.Error("render dashboard cards", "err", err)
	}
}
