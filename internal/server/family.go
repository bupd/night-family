package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bupd/night-family/internal/family"
)

func decodeMember(r *http.Request) (family.Member, error) {
	var m family.Member
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		return m, err
	}
	return m, nil
}

// familyRoutes registers the /api/v1/family* endpoints on the given mux.
// Wired only when cfg.Family is non-nil.
func (s *Server) familyRoutes(mux *http.ServeMux) {
	if s.cfg.Family == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/family", s.listFamily)
	mux.HandleFunc("GET /api/v1/family/{name}", s.getFamilyMember)
	mux.HandleFunc("POST /api/v1/family", s.createFamilyMember)
	mux.HandleFunc("PUT /api/v1/family/{name}", s.replaceFamilyMember)
	mux.HandleFunc("DELETE /api/v1/family/{name}", s.deleteFamilyMember)
	mux.HandleFunc("POST /api/v1/family/validate", s.validateFamilyMember)
	mux.HandleFunc("GET /family", s.familyPage)
}

func (s *Server) createFamilyMember(w http.ResponseWriter, r *http.Request) {
	m, err := decodeMember(r)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "bad_request", "invalid JSON body: "+err.Error(), r.URL.Path)
		return
	}
	stored, err := s.cfg.Family.Add(m)
	if err != nil {
		var ve family.ValidationError
		if errors.As(err, &ve) {
			writeValidation(w, http.StatusBadRequest, ve, r.URL.Path)
			return
		}
		if errors.Is(err, family.ErrDuplicate) {
			writeProblem(w, http.StatusConflict, "duplicate", err.Error(), r.URL.Path)
			return
		}
		writeProblem(w, http.StatusInternalServerError, "internal_error", err.Error(), r.URL.Path)
		return
	}
	writeJSON(w, http.StatusCreated, stored)
}

func (s *Server) replaceFamilyMember(w http.ResponseWriter, r *http.Request) {
	m, err := decodeMember(r)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "bad_request", "invalid JSON body: "+err.Error(), r.URL.Path)
		return
	}
	name := r.PathValue("name")
	if m.Name == "" {
		m.Name = name
	}
	if m.Name != name {
		writeProblem(w, http.StatusBadRequest, "bad_request",
			"body.name does not match URL name", r.URL.Path)
		return
	}
	stored, err := s.cfg.Family.Put(m)
	if err != nil {
		var ve family.ValidationError
		if errors.As(err, &ve) {
			writeValidation(w, http.StatusBadRequest, ve, r.URL.Path)
			return
		}
		writeProblem(w, http.StatusInternalServerError, "internal_error", err.Error(), r.URL.Path)
		return
	}
	writeJSON(w, http.StatusOK, stored)
}

func (s *Server) deleteFamilyMember(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	err := s.cfg.Family.Remove(name)
	if errors.Is(err, family.ErrNotFound) {
		writeProblem(w, http.StatusNotFound, "not_found", "Family member not found", r.URL.Path)
		return
	}
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal_error", err.Error(), r.URL.Path)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) validateFamilyMember(w http.ResponseWriter, r *http.Request) {
	m, err := decodeMember(r)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "bad_request", "invalid JSON body: "+err.Error(), r.URL.Path)
		return
	}
	m.ApplyDefaults()
	issues := family.Validate(m)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     len(issues) == 0,
		"errors": issues,
	})
}

func writeValidation(w http.ResponseWriter, status int, ve family.ValidationError, instance string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":     "https://night-family.dev/errors/validation",
		"title":    "Invalid family member",
		"status":   status,
		"detail":   ve.Error(),
		"instance": instance,
		"errors":   ve,
	})
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
