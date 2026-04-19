package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bupd/night-family/internal/duty"
	"github.com/bupd/night-family/internal/family"
	"github.com/bupd/night-family/internal/schedule"
)

func newTestServerWithPlanner(t *testing.T) *Server {
	t.Helper()
	fam := family.NewStore()
	defaults, err := family.LoadDefaults()
	if err != nil {
		t.Fatalf("LoadDefaults: %v", err)
	}
	fam.Seed(defaults)
	sched := schedule.Default()
	sched.TimeZone = "Europe/Berlin"
	loc, _ := time.LoadLocation("Europe/Berlin")
	fakeNow := time.Date(2026, 4, 17, 23, 0, 0, 0, loc)
	s, err := New(Config{
		Addr:     "127.0.0.1:0",
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Family:   fam,
		Duties:   duty.NewBuiltinRegistry(),
		Schedule: &sched,
		Clock:    func() time.Time { return fakeNow },
	})
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	return s
}

func TestPreviewNightJSON(t *testing.T) {
	s := newTestServerWithPlanner(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nights/preview", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var plan map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &plan); err != nil {
		t.Fatalf("decode: %v", err)
	}
	slots, ok := plan["slots"].([]any)
	if !ok || len(slots) == 0 {
		t.Fatalf("no slots in preview: %v", plan)
	}
	if plan["reserved_tokens"].(float64) <= 0 {
		t.Errorf("reserved_tokens not positive")
	}
}

func TestPreviewNightHonoursBudget(t *testing.T) {
	s := newTestServerWithPlanner(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nights/preview?budget=10000", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var plan map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &plan)
	if plan["reserved_tokens"].(float64) > 10000 {
		t.Errorf("reserved_tokens %v > 10000", plan["reserved_tokens"])
	}
}

func TestPlanPageRenders(t *testing.T) {
	s := newTestServerWithPlanner(t)
	req := httptest.NewRequest(http.MethodGet, "/plan", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{"Tonight", "Member", "Est. tokens", "rick", "morty"} {
		if !strings.Contains(body, want) {
			t.Errorf("plan page missing %q", want)
		}
	}
}
