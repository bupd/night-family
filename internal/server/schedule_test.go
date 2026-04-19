package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bupd/night-family/internal/schedule"
)

func TestGetSchedule(t *testing.T) {
	sc := schedule.Schedule{WindowStart: "22:00", WindowEnd: "05:00", TimeZone: "Europe/Berlin"}
	loc, _ := time.LoadLocation("Europe/Berlin")
	fakeNow := time.Date(2026, 4, 17, 23, 0, 0, 0, loc)
	s, err := New(Config{
		Addr:     "127.0.0.1:0",
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Schedule: &sc,
		Clock:    func() time.Time { return fakeNow },
	})
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedule", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["window_start"] != "22:00" {
		t.Errorf("window_start = %v, want 22:00", body["window_start"])
	}
	if body["in_window"] != true {
		t.Errorf("in_window = %v, want true", body["in_window"])
	}
}

func TestGetScheduleAbsentWhenNoSchedule(t *testing.T) {
	s, err := New(Config{
		Addr:   "127.0.0.1:0",
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedule", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}
