package app_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tinh-tien-api/internal/app"
	"tinh-tien-api/internal/pkg/httputil"
)

func TestNotFoundUsesUnifiedResponse(t *testing.T) {
	r := app.NewRouter(app.Handlers{})
	req := httptest.NewRequest(http.MethodGet, "/v1/unknown", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d", w.Code)
	}

	var resp httputil.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Success {
		t.Fatal("expected success false")
	}
	if resp.Data != nil {
		t.Fatal("expected nil data")
	}
	if resp.Pagination != nil {
		t.Fatal("expected nil pagination")
	}
}

func TestHealthzUsesUnifiedResponse(t *testing.T) {
	r := app.NewRouter(app.Handlers{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp httputil.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Success || resp.Message == "" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}
