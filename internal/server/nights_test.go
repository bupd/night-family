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
	"github.com/bupd/night-family/internal/schedule"
	"github.com/bupd/night-family/internal/storage"
)

func newTestServerWithNight(t *testing.T) *Server {
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
	sched := schedule.Default()
	s, err := New(Config{
		Addr:     "127.0.0.1:0",
		Logger:   logger,
		Family:   fam,
		Duties:   duty.NewBuiltinRegistry(),
		Storage:  db,
		Runner:   rn,
		Schedule: &sched,
	})
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	return s
}

func TestTriggerNightOnlyOneMember(t *testing.T) {
	s := newTestServerWithNight(t)
	body, _ := json.Marshal(map[string]any{"only_members": []string{"jerry"}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nights/trigger", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	var res map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	runs, _ := res["runs"].([]any)
	if len(runs) == 0 {
		t.Fatalf("no runs in night result")
	}
	for _, r := range runs {
		m := r.(map[string]any)
		if m["member"] != "jerry" {
			t.Errorf("run member = %v, want jerry", m["member"])
		}
	}
}

func TestTriggerNightDryRunPersistsNightButNoRuns(t *testing.T) {
	s := newTestServerWithNight(t)
	body, _ := json.Marshal(map[string]any{"dry_run": true})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nights/trigger", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d", rr.Code)
	}
	var res map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &res)
	runs, _ := res["runs"].([]any)
	if len(runs) != 0 {
		t.Errorf("dry_run produced %d runs, want 0", len(runs))
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/nights", nil)
	listRR := httptest.NewRecorder()
	s.routes().ServeHTTP(listRR, listReq)
	var listBody map[string]any
	_ = json.Unmarshal(listRR.Body.Bytes(), &listBody)
	items, _ := listBody["items"].([]any)
	if len(items) != 1 {
		t.Errorf("nights list = %d items, want 1", len(items))
	}
}

func TestGetNightNotFound(t *testing.T) {
	s := newTestServerWithNight(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nights/night_missing", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}
