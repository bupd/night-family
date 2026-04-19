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

func TestGetNightDigest(t *testing.T) {
	db, err := storage.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	started := time.Now().UTC()
	nightID := "night_01HDIGEST00000000000000AA"
	if err := db.InsertNight(ctx, storage.Night{ID: nightID, StartedAt: started}); err != nil {
		t.Fatalf("InsertNight: %v", err)
	}
	id := nightID
	if err := db.InsertRun(ctx, storage.Run{
		ID:      "run_01HDIGESTRUN0000000000000A",
		NightID: &id,
		Member:  "jerry", Duty: "lint-fix",
		Status:    storage.RunSucceeded,
		StartedAt: started,
	}); err != nil {
		t.Fatalf("InsertRun: %v", err)
	}
	_ = db.FinishNight(ctx, nightID, started.Add(5*time.Minute), "done")

	s, _ := New(Config{
		Addr:    "127.0.0.1:0",
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		Storage: db,
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nights/"+nightID+"/digest", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if !strings.HasPrefix(rr.Header().Get("Content-Type"), "text/markdown") {
		t.Errorf("content-type = %q", rr.Header().Get("Content-Type"))
	}
	for _, want := range []string{"# ", "Night " + nightID, "`jerry`", "`lint-fix`", "succeeded"} {
		if !strings.Contains(rr.Body.String(), want) {
			t.Errorf("digest missing %q\n--- body ---\n%s", want, rr.Body.String())
		}
	}
}

func TestGetNightDigestNotFound(t *testing.T) {
	db, _ := storage.Open(context.Background(), ":memory:")
	t.Cleanup(func() { _ = db.Close() })
	s, _ := New(Config{
		Addr:    "127.0.0.1:0",
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		Storage: db,
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nights/night_missing/digest", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rr.Code)
	}
}
