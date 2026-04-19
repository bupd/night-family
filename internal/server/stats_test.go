package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bupd/night-family/internal/storage"
)

func TestStatsEndpointEmpty(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	var st map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &st)
	if st["nights"].(float64) != 0 || st["runs"].(float64) != 0 || st["prs"].(float64) != 0 {
		t.Errorf("non-zero on empty DB: %v", st)
	}
}

func TestStatsWithRows(t *testing.T) {
	db, _ := storage.Open(context.Background(), ":memory:")
	t.Cleanup(func() { _ = db.Close() })
	_ = db.InsertNight(context.Background(), storage.Night{
		ID: "night_01HX00000000000000000000AA", StartedAt: time.Now(),
	})
	_ = db.InsertRun(context.Background(), storage.Run{
		ID: "run_01HX00000000000000000000BB", Member: "jerry", Duty: "lint-fix",
		Status: storage.RunSucceeded, StartedAt: time.Now(),
	})
	s, _ := New(Config{
		Addr: "127.0.0.1:0", Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Storage: db,
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	var st map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &st)
	if st["nights"].(float64) != 1 || st["runs"].(float64) != 1 {
		t.Errorf("counts off: %v", st)
	}
	byStatus := st["runs_by_status"].(map[string]any)
	if byStatus["succeeded"].(float64) != 1 {
		t.Errorf("succeeded = %v", byStatus["succeeded"])
	}
}
