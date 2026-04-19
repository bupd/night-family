package server

import (
	"context"
	"net/http"
	"time"

	"github.com/bupd/night-family/internal/storage"
)

func (s *Server) nightsPageRoutes(mux *http.ServeMux) {
	if s.cfg.Storage == nil {
		return
	}
	mux.HandleFunc("GET /nights", s.nightsPage)
}

func (s *Server) nightsPage(w http.ResponseWriter, r *http.Request) {
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
		Title:  "Nights — night-family",
		Nights: nights,
	}
	s.renderPage(w, "nights", data)
}
