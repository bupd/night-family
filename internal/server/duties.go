package server

import (
	"net/http"

	"github.com/bupd/night-family/internal/duty"
)

func (s *Server) dutiesRoutes(mux *http.ServeMux) {
	if s.cfg.Duties == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/duties", s.listDuties)
	mux.HandleFunc("GET /api/v1/duties/{type}", s.getDuty)
	mux.HandleFunc("GET /duties", s.dutiesPage)
}

func (s *Server) listDuties(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.cfg.Duties.List())
}

func (s *Server) getDuty(w http.ResponseWriter, r *http.Request) {
	typ := r.PathValue("type")
	info, ok := s.cfg.Duties.Get(typ)
	if !ok {
		writeProblem(w, http.StatusNotFound, "not_found", "Duty type not found", r.URL.Path)
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (s *Server) dutiesPage(w http.ResponseWriter, _ *http.Request) {
	all := s.cfg.Duties.List()
	// Group by Output for the card layout.
	byOutput := map[duty.Output][]duty.Info{}
	for _, d := range all {
		byOutput[d.Output] = append(byOutput[d.Output], d)
	}
	data := struct {
		Title     string
		Version   string
		Duties    []duty.Info
		ByOutput  map[duty.Output][]duty.Info
		OutputPR  duty.Output
		OutputIss duty.Output
		OutputIPR duty.Output
		OutputNot duty.Output
	}{
		Title:     "Duties — night-family",
		Duties:    all,
		ByOutput:  byOutput,
		OutputPR:  duty.OutputPR,
		OutputIss: duty.OutputIssue,
		OutputIPR: duty.OutputIssuePlusPR,
		OutputNot: duty.OutputNote,
	}
	s.renderPage(w, "duties", data)
}
