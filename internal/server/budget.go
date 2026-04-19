package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/bupd/night-family/internal/storage"
)

func (s *Server) budgetRoutes(mux *http.ServeMux) {
	if s.cfg.Storage == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/budget", s.getBudget)
}

func (s *Server) getBudget(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	snap, err := s.cfg.Storage.LatestBudgetSnapshot(ctx)
	if errors.Is(err, storage.ErrNotFound) {
		// No snapshots yet — return an empty shape rather than 404 so
		// dashboard code doesn't have to special-case.
		writeJSON(w, http.StatusOK, storage.BudgetSnapshot{
			TakenAt:    time.Now().UTC(),
			Provider:   "unknown",
			Confidence: "low",
		})
		return
	}
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "db_error", err.Error(), r.URL.Path)
		return
	}
	writeJSON(w, http.StatusOK, snap)
}
