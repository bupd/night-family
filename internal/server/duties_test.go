package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bupd/night-family/internal/duty"
)

func newTestServerWithDuties(t *testing.T) *Server {
	t.Helper()
	s, err := New(Config{
		Addr:   "127.0.0.1:0",
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Duties: duty.NewBuiltinRegistry(),
	})
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	return s
}

func TestListDuties(t *testing.T) {
	s := newTestServerWithDuties(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/duties", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var list []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) < 18 {
		t.Fatalf("duty count = %d, want >= 18", len(list))
	}
	types := map[string]bool{}
	for _, d := range list {
		types[d["type"].(string)] = true
	}
	for _, want := range []string{"vuln-scan", "lint-fix", "docs-drift", "todo-triage"} {
		if !types[want] {
			t.Errorf("duty catalogue missing %q", want)
		}
	}
}

func TestGetDutyHit(t *testing.T) {
	s := newTestServerWithDuties(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/duties/lint-fix", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var info map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &info); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if info["type"] != "lint-fix" {
		t.Fatalf("type = %v, want lint-fix", info["type"])
	}
}

func TestGetDutyMiss(t *testing.T) {
	s := newTestServerWithDuties(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/duties/does-not-exist", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("content-type = %q, want application/problem+json", ct)
	}
}

func TestDutiesPageRenders(t *testing.T) {
	s := newTestServerWithDuties(t)
	req := httptest.NewRequest(http.MethodGet, "/duties", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{"vuln-scan", "lint-fix", "Duties"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}
}
