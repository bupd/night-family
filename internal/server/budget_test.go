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

func TestGetBudgetEmpty(t *testing.T) {
	db, err := storage.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	s, _ := New(Config{
		Addr:    "127.0.0.1:0",
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		Storage: db,
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/budget", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	var snap map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &snap)
	if snap["provider"] != "unknown" {
		t.Errorf("provider = %v, want unknown", snap["provider"])
	}
}

func TestGetBudgetWithSnapshot(t *testing.T) {
	db, _ := storage.Open(context.Background(), ":memory:")
	t.Cleanup(func() { _ = db.Close() })
	_, err := db.InsertBudgetSnapshot(context.Background(), storage.BudgetSnapshot{
		Provider:                "claude",
		RemainingTokensEstimate: 42000,
		ReservedForTonight:      10000,
		TakenAt:                 time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	s, _ := New(Config{
		Addr: "127.0.0.1:0", Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Storage: db,
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/budget", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	var snap map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &snap)
	if snap["provider"] != "claude" {
		t.Errorf("provider = %v", snap["provider"])
	}
	if snap["remaining_tokens_estimate"].(float64) != 42000 {
		t.Errorf("remaining = %v", snap["remaining_tokens_estimate"])
	}
}
