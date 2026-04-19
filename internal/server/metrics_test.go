package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bupd/night-family/internal/storage"
)

func TestMetricsAlwaysServed(t *testing.T) {
	s, _ := New(Config{
		Addr:   "127.0.0.1:0",
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("content-type = %q", ct)
	}
	for _, want := range []string{"nf_up 1", "nf_uptime_seconds", "# HELP"} {
		if !strings.Contains(rr.Body.String(), want) {
			t.Errorf("body missing %q", want)
		}
	}
}

func TestMetricsIncludesStorageCounts(t *testing.T) {
	db, err := storage.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_ = db.InsertRun(context.Background(), storage.Run{
		ID: "run_01HMETRICS0000000000000000", Member: "rick", Duty: "vuln-scan",
		Status: storage.RunSucceeded, StartedAt: time.Now(),
	})
	s, _ := New(Config{
		Addr: "127.0.0.1:0", Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Storage: db,
	})
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	body := rr.Body.String()
	for _, want := range []string{
		"nf_runs_total 1",
		`nf_runs_by_status{status="succeeded"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q\n--- full body ---\n%s", want, body)
		}
	}
}
