package server

import (
	"context"
	"net/http"
	"time"

	"github.com/bupd/night-family/internal/provider"
)

func (s *Server) providerRoutes(mux *http.ServeMux) {
	if s.cfg.Provider == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/provider", s.getProviderStatus)
}

func (s *Server) getProviderStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	out := map[string]any{
		"name": s.cfg.Provider.Name(),
	}
	if p, ok := s.cfg.Provider.(provider.StatusProber); ok {
		st, err := p.SessionStatus(ctx)
		if err == nil {
			out["status"] = st
		} else {
			out["status_error"] = err.Error()
		}
	} else {
		out["status_supported"] = false
	}
	writeJSON(w, http.StatusOK, out)
}
