package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bupd/night-family/internal/family"
)

// familyRoutes registers the /api/v1/family* endpoints on the given mux.
// Wired only when cfg.Family is non-nil.
func (s *Server) familyRoutes(mux *http.ServeMux) {
	if s.cfg.Family == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/family", s.listFamily)
	mux.HandleFunc("GET /api/v1/family/{name}", s.getFamilyMember)
	mux.HandleFunc("GET /family", s.familyPage)
}

func (s *Server) listFamily(w http.ResponseWriter, _ *http.Request) {
	members := s.cfg.Family.List()
	writeJSON(w, http.StatusOK, members)
}

func (s *Server) getFamilyMember(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	m, err := s.cfg.Family.Get(name)
	if errors.Is(err, family.ErrNotFound) {
		writeProblem(w, http.StatusNotFound, "not_found", "Family member not found", r.URL.Path)
		return
	}
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal_error", err.Error(), r.URL.Path)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (s *Server) familyPage(w http.ResponseWriter, _ *http.Request) {
	members := s.cfg.Family.List()
	data := struct {
		Title   string
		Version string
		Members []family.Member
	}{
		Title:   "Family — night-family",
		Version: "",
		Members: members,
	}
	s.renderPage(w, "family", data)
}

func writeProblem(w http.ResponseWriter, status int, slug, detail, instance string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":     "https://night-family.dev/errors/" + slug,
		"title":    http.StatusText(status),
		"status":   status,
		"detail":   detail,
		"instance": instance,
	})
}
