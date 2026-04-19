package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOpenAPIYAMLServed(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/yaml") {
		t.Fatalf("content-type = %q, want application/yaml…", ct)
	}
	if !strings.Contains(rr.Body.String(), "openapi: 3.1.0") {
		t.Fatalf("spec does not declare openapi: 3.1.0")
	}
	// Ensure the embedded YAML is valid YAML.
	var v any
	if err := yaml.Unmarshal(rr.Body.Bytes(), &v); err != nil {
		t.Fatalf("embedded openapi.yaml is not valid YAML: %v", err)
	}
}

func TestOpenAPIJSONServed(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content-type = %q, want application/json", ct)
	}
	var doc map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got, _ := doc["openapi"].(string); got != "3.1.0" {
		t.Fatalf("openapi field = %q, want 3.1.0", got)
	}
	info, ok := doc["info"].(map[string]any)
	if !ok {
		t.Fatalf("info object missing")
	}
	if title, _ := info["title"].(string); !strings.Contains(title, "night-family") {
		t.Fatalf("info.title = %q, want contains 'night-family'", title)
	}
	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatalf("paths object missing")
	}
	for _, want := range []string{"/healthz", "/family", "/runs", "/budget", "/schedule", "/nights/current"} {
		if _, exists := paths[want]; !exists {
			t.Errorf("spec missing path %q", want)
		}
	}
}

func TestDocsPageServed(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{"swagger-ui", "/openapi.yaml"} {
		if !strings.Contains(body, want) {
			t.Errorf("docs body missing %q", want)
		}
	}
}
