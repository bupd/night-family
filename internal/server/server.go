// Package server wires up the HTTP surface served by nfd: the JSON API
// under /api/v1, the health endpoints, and the HTMX-rendered UI.
//
// In this skeleton only /healthz, /readyz, /version, and a minimal / page
// are wired. Subsequent iterations will mount the API and UI handlers from
// their own packages.
package server

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/bupd/night-family/internal/version"
)

//go:embed all:web
var webFS embed.FS

// WebFS returns the embedded UI assets so tests and tools can inspect them.
func WebFS() fs.FS {
	sub, _ := fs.Sub(webFS, "web")
	return sub
}

// Config controls how the server binds and renders.
type Config struct {
	// Addr is the listen address, e.g. "127.0.0.1:7337".
	Addr string
	// Logger used by handlers. Must be non-nil.
	Logger *slog.Logger
}

// Server is the running HTTP server.
type Server struct {
	cfg   Config
	srv   *http.Server
	pages map[string]*template.Template
	web   fs.FS
	spec  *specBundle
}

// New constructs a Server. Templates are parsed eagerly; if parsing fails
// the caller gets an error instead of a half-initialised server.
func New(cfg Config) (*Server, error) {
	if cfg.Logger == nil {
		return nil, errors.New("server: cfg.Logger is required")
	}
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:7337"
	}
	web := WebFS()
	pages, err := parsePages(web)
	if err != nil {
		return nil, err
	}
	spec, err := loadSpec()
	if err != nil {
		return nil, err
	}
	s := &Server{cfg: cfg, pages: pages, web: web, spec: spec}
	s.srv = &http.Server{
		Addr:              cfg.Addr,
		Handler:           s.routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	return s, nil
}

// Addr returns the configured listen address.
func (s *Server) Addr() string { return s.cfg.Addr }

// ListenAndServe starts the server. It blocks until the server stops.
func (s *Server) ListenAndServe() error {
	s.cfg.Logger.Info("server listening", "addr", s.cfg.Addr)
	if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully drains the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /readyz", s.readyz)
	mux.HandleFunc("GET /version", s.versionJSON)
	mux.HandleFunc("GET /openapi.yaml", s.serveOpenAPIYAML)
	mux.HandleFunc("GET /openapi.json", s.serveOpenAPIJSON)
	mux.HandleFunc("GET /docs", s.serveDocs)
	mux.HandleFunc("GET /", s.index)

	if staticSub, err := fs.Sub(s.web, "static"); err == nil {
		mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticSub)))
	}

	return logMiddleware(s.cfg.Logger, mux)
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) readyz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) versionJSON(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, version.Current())
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data := struct {
		Title   string
		Version string
	}{
		Title:   "night-family",
		Version: version.Current().Version,
	}
	s.renderPage(w, "index", data)
}

// renderPage executes a named page's template set. Each page is a fresh
// template cloned at startup so sibling pages don't accidentally
// override each other's `{{define "content"}}` blocks.
func (s *Server) renderPage(w http.ResponseWriter, page string, data any) {
	tpl, ok := s.pages[page]
	if !ok {
		http.Error(w, "template not found: "+page, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, "base", data); err != nil {
		s.cfg.Logger.Error("render page", "page", page, "err", err)
	}
}

// parsePages returns one fully-populated template per page. Each page
// gets base.html.tmpl plus its own file parsed in isolation so the
// `content` block isn't shared across pages.
func parsePages(web fs.FS) (map[string]*template.Template, error) {
	pages := map[string]string{
		"index": "templates/index.html.tmpl",
		"docs":  "templates/docs.html.tmpl",
	}
	out := make(map[string]*template.Template, len(pages))
	for name, path := range pages {
		tpl, err := template.New(name).ParseFS(web, "templates/base.html.tmpl", path)
		if err != nil {
			return nil, err
		}
		out[name] = tpl
	}
	return out, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func logMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		logger.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"dur", time.Since(start).String(),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
