package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bupd/night-family/internal/family"
)

func newTestServerBlankFamily(t *testing.T) (*Server, *family.Store) {
	t.Helper()
	fam := family.NewStore()
	s, err := New(Config{
		Addr:   "127.0.0.1:0",
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Family: fam,
	})
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	return s, fam
}

func postJSON(t *testing.T, s *Server, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	return rr
}

func TestCreateMember(t *testing.T) {
	s, fam := newTestServerBlankFamily(t)
	member := map[string]any{
		"name":          "custom",
		"role":          "custom role",
		"system_prompt": "you are custom",
	}
	rr := postJSON(t, s, http.MethodPost, "/api/v1/family", member)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if _, err := fam.Get("custom"); err != nil {
		t.Fatalf("member not stored: %v", err)
	}
}

func TestCreateMemberDuplicate(t *testing.T) {
	s, _ := newTestServerBlankFamily(t)
	body := map[string]any{"name": "dup", "role": "r", "system_prompt": "p"}
	_ = postJSON(t, s, http.MethodPost, "/api/v1/family", body)
	rr := postJSON(t, s, http.MethodPost, "/api/v1/family", body)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rr.Code)
	}
}

func TestCreateMemberInvalid(t *testing.T) {
	s, _ := newTestServerBlankFamily(t)
	body := map[string]any{"name": "BAD-CASE", "role": "r"} // missing system_prompt + bad name
	rr := postJSON(t, s, http.MethodPost, "/api/v1/family", body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("content-type = %q", ct)
	}
}

func TestReplaceMember(t *testing.T) {
	s, fam := newTestServerBlankFamily(t)
	_ = postJSON(t, s, http.MethodPost, "/api/v1/family",
		map[string]any{"name": "alice", "role": "old", "system_prompt": "p"})
	rr := postJSON(t, s, http.MethodPut, "/api/v1/family/alice",
		map[string]any{"name": "alice", "role": "new", "system_prompt": "p2"})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	got, _ := fam.Get("alice")
	if got.Role != "new" {
		t.Errorf("role = %q, want new", got.Role)
	}
}

func TestReplaceMemberMismatchedName(t *testing.T) {
	s, _ := newTestServerBlankFamily(t)
	rr := postJSON(t, s, http.MethodPut, "/api/v1/family/alice",
		map[string]any{"name": "bob", "role": "r", "system_prompt": "p"})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestDeleteMember(t *testing.T) {
	s, fam := newTestServerBlankFamily(t)
	_ = postJSON(t, s, http.MethodPost, "/api/v1/family",
		map[string]any{"name": "temp", "role": "r", "system_prompt": "p"})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/family/temp", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rr.Code)
	}
	if _, err := fam.Get("temp"); err == nil {
		t.Errorf("expected ErrNotFound after delete")
	}
}

func TestDeleteMemberMissing(t *testing.T) {
	s, _ := newTestServerBlankFamily(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/family/ghost", nil)
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestValidateMember(t *testing.T) {
	s, _ := newTestServerBlankFamily(t)
	rr := postJSON(t, s, http.MethodPost, "/api/v1/family/validate",
		map[string]any{"name": "ok", "role": "r", "system_prompt": "p"})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["ok"] != true {
		t.Errorf("ok = %v, want true", body["ok"])
	}

	rr = postJSON(t, s, http.MethodPost, "/api/v1/family/validate",
		map[string]any{"name": "BAD"}) // missing role + system_prompt
	if rr.Code != http.StatusOK {
		t.Fatalf("validate status = %d", rr.Code)
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["ok"] != false {
		t.Errorf("ok = %v, want false", body["ok"])
	}
}
