package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bupd/night-family/internal/duty"
	"github.com/bupd/night-family/internal/family"
	"github.com/bupd/night-family/internal/provider"
	"github.com/bupd/night-family/internal/runner"
	"github.com/bupd/night-family/internal/storage"
)

func newTestServerWithRunner(t *testing.T) *Server {
	t.Helper()
	fam := family.NewStore()
	defaults, _ := family.LoadDefaults()
	fam.Seed(defaults)
	db, err := storage.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rn, err := runner.New(runner.Deps{
		Family:   fam,
		Duties:   duty.NewBuiltinRegistry(),
		Storage:  db,
		Provider: provider.NewMock(),
		Logger:   logger,
	})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	s, err := New(Config{
		Addr:    "127.0.0.1:0",
		Logger:  logger,
		Family:  fam,
		Duties:  duty.NewBuiltinRegistry(),
		Storage: db,
		Runner:  rn,
	})
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	return s
}

func TestCreateRunSucceeds(t *testing.T) {
	s := newTestServerWithRunner(t)
	body, _ := json.Marshal(map[string]string{"member": "jerry", "duty": "lint-fix"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	var run map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &run); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if run["status"] != "succeeded" {
		t.Errorf("status = %v, want succeeded", run["status"])
	}
	if run["member"] != "jerry" {
		t.Errorf("member = %v, want jerry", run["member"])
	}
}

func TestCreateRunRejectsMissingFields(t *testing.T) {
	s := newTestServerWithRunner(t)
	body, _ := json.Marshal(map[string]string{"member": "jerry"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestCreateRunUnknownMember(t *testing.T) {
	s := newTestServerWithRunner(t)
	body, _ := json.Marshal(map[string]string{"member": "ghost", "duty": "lint-fix"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}
