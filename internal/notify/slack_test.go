package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSlackPostsJSON(t *testing.T) {
	var got map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("content-type = %q", ct)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	s := NewSlack(srv.URL)
	s.Client = srv.Client()
	if err := s.Notify(context.Background(), "title", "body"); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if got["text"] != "title\n\nbody" {
		t.Errorf("text = %q", got["text"])
	}
}

func TestSlackTruncatesLongBody(t *testing.T) {
	var got map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	s := NewSlack(srv.URL)
	s.Client = srv.Client()
	long := strings.Repeat("x", 5000)
	_ = s.Notify(context.Background(), "t", long)
	if !strings.HasSuffix(got["text"], "(truncated)") {
		t.Errorf("expected truncation marker, got tail %q", got["text"][len(got["text"])-20:])
	}
}

func TestSlackNoURLIsNoop(t *testing.T) {
	s := NewSlack("")
	if err := s.Notify(context.Background(), "t", "b"); err != nil {
		t.Fatalf("empty URL should no-op, got %v", err)
	}
}

func TestSlackSurfacesNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)
	s := NewSlack(srv.URL)
	s.Client = srv.Client()
	if err := s.Notify(context.Background(), "t", "b"); err == nil {
		t.Fatalf("expected error on 400")
	}
}

func TestNoop(t *testing.T) {
	if err := (Noop{}).Notify(context.Background(), "t", "b"); err != nil {
		t.Errorf("Noop.Notify = %v, want nil", err)
	}
}
