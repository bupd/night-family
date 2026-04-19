package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bupd/night-family/internal/storage"
)

func newTestServerWithStorage(t *testing.T) (*Server, *storage.DB) {
	t.Helper()
	db, err := storage.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	s, err := New(Config{
		Addr:    "127.0.0.1:0",
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		Storage: db,
	})
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	return s, db
}

func TestListRunsEmpty(t *testing.T) {
	s, _ := newTestServerWithStorage(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["items"] == nil {
		t.Fatalf("missing items field")
	}
}

func TestListAndGetRun(t *testing.T) {
	s, db := newTestServerWithStorage(t)
	run := storage.Run{
		ID:        "run_01HTEST0000000000000000XY",
		Member:    "rick",
		Duty:      "vuln-scan",
		Status:    storage.RunSucceeded,
		StartedAt: time.Date(2026, 4, 17, 22, 0, 0, 0, time.UTC),
	}
	if err := db.InsertRun(context.Background(), run); err != nil {
		t.Fatalf("InsertRun: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list status = %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "rick") {
		t.Fatalf("rick not in list body: %s", rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+run.ID, nil)
	rr = httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get status = %d", rr.Code)
	}
	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["id"] != run.ID {
		t.Errorf("id = %v, want %v", got["id"], run.ID)
	}
}

func TestGetRunNotFound(t *testing.T) {
	s, _ := newTestServerWithStorage(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run_missing", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("content-type = %q", ct)
	}
}

func TestRunsPageEmpty(t *testing.T) {
	s, _ := newTestServerWithStorage(t)
	req := httptest.NewRequest(http.MethodGet, "/runs", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Runs") {
		t.Errorf("page missing title")
	}
}
