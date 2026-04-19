package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bupd/night-family/internal/family"
)

func newTestServerWithFamily(t *testing.T) *Server {
	t.Helper()
	fam := family.NewStore()
	defaults, err := family.LoadDefaults()
	if err != nil {
		t.Fatalf("LoadDefaults: %v", err)
	}
	fam.Seed(defaults)
	s, err := New(Config{
		Addr:   "127.0.0.1:0",
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Family: fam,
	})
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	return s
}

func TestListFamily(t *testing.T) {
	s := newTestServerWithFamily(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/family", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var members []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &members); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(members) < 7 {
		t.Fatalf("member count = %d, want >= 7", len(members))
	}
	names := make(map[string]bool)
	for _, m := range members {
		names[m["name"].(string)] = true
	}
	for _, expected := range []string{"rick", "morty", "summer", "beth", "jerry", "birdperson", "squanchy"} {
		if !names[expected] {
			t.Errorf("missing %q in /api/v1/family response", expected)
		}
	}
}

func TestGetFamilyMemberFound(t *testing.T) {
	s := newTestServerWithFamily(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/family/rick", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var m map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if m["name"] != "rick" {
		t.Fatalf("name = %v, want rick", m["name"])
	}
}

func TestGetFamilyMemberNotFound(t *testing.T) {
	s := newTestServerWithFamily(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/family/ghost", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if ct != "application/problem+json" {
		t.Fatalf("content-type = %q, want application/problem+json", ct)
	}
}

func TestFamilyPageRenders(t *testing.T) {
	s := newTestServerWithFamily(t)
	req := httptest.NewRequest(http.MethodGet, "/family", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{"rick", "morty", "summer", "Family", "member"} {
		if !strings.Contains(body, want) {
			t.Errorf("family page missing %q", want)
		}
	}
}

func TestFamilyRoutesDisabledWhenStoreNil(t *testing.T) {
	s, err := New(Config{
		Addr:   "127.0.0.1:0",
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/family", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (routes should be absent)", rr.Code)
	}
}
