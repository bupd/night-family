package server

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"

	"gopkg.in/yaml.v3"
)

// apiFS bundles the OpenAPI spec next to the server package so the daemon
// can serve it at /openapi.yaml and /openapi.json without an external
// filesystem dependency.
//
//go:embed all:apiassets
var apiFS embed.FS

type specBundle struct {
	yaml []byte
	json []byte
}

func loadSpec() (*specBundle, error) {
	sub, err := fs.Sub(apiFS, "apiassets")
	if err != nil {
		return nil, err
	}
	yamlBytes, err := fs.ReadFile(sub, "openapi.yaml")
	if err != nil {
		return nil, fmt.Errorf("read embedded openapi.yaml: %w", err)
	}
	var v any
	if err := yaml.Unmarshal(yamlBytes, &v); err != nil {
		return nil, fmt.Errorf("parse embedded openapi.yaml: %w", err)
	}
	jsonBytes, err := marshalYAMLAsJSON(v)
	if err != nil {
		return nil, fmt.Errorf("re-encode openapi spec as json: %w", err)
	}
	return &specBundle{yaml: yamlBytes, json: jsonBytes}, nil
}

// marshalYAMLAsJSON converts the decoded YAML tree into JSON. yaml.v3
// hands us `map[string]any` for mapping nodes (not `map[any]any` as v2
// did), so we can encode directly — but we still normalise any
// `map[any]any` that leaks in from nested docs.
func marshalYAMLAsJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(normalise(v)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func normalise(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, vv := range t {
			out[k] = normalise(vv)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, vv := range t {
			out[fmt.Sprint(k)] = normalise(vv)
		}
		return out
	case []any:
		for i := range t {
			t[i] = normalise(t[i])
		}
		return t
	default:
		return v
	}
}

func (s *Server) serveOpenAPIYAML(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	_, _ = w.Write(s.spec.yaml)
}

func (s *Server) serveOpenAPIJSON(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(s.spec.json)
}

func (s *Server) serveDocs(w http.ResponseWriter, _ *http.Request) {
	data := struct {
		Title   string
		Version string
	}{
		Title:   "API docs — night-family",
		Version: "",
	}
	s.renderPage(w, "docs", data)
}
