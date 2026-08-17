package httputil_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tinh-tien-api/internal/pkg/httputil"
)

func TestOKResponseEnvelope(t *testing.T) {
	w := httptest.NewRecorder()
	httputil.OK(w, "items retrieved", map[string]string{"id": "1"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var resp httputil.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success true")
	}
	if resp.Message != "items retrieved" {
		t.Fatalf("message = %q", resp.Message)
	}
	if resp.Error != nil {
		t.Fatal("expected nil error")
	}
	if resp.Pagination != nil {
		t.Fatal("expected nil pagination")
	}
}

func TestEmptySliceNormalized(t *testing.T) {
	w := httptest.NewRecorder()
	var items []string
	httputil.OK(w, "ok", items)

	var resp httputil.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	raw, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	if string(raw) != "[]" {
		t.Fatalf("expected empty array, got %s", raw)
	}
}

func TestFailResponseEnvelope(t *testing.T) {
	w := httptest.NewRecorder()
	httputil.Fail(w, http.StatusBadRequest, "invalid input", "field required")

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
	if resp.Error == nil {
		t.Fatal("expected error detail")
	}
}

func TestPaginationResponse(t *testing.T) {
	w := httptest.NewRecorder()
	pag := httputil.NewPagination(1, 20, 45)
	httputil.OKWithPagination(w, "ok", []string{"a"}, pag)

	var resp httputil.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Pagination == nil || resp.Pagination.Total != 45 || resp.Pagination.TotalPages != 3 {
		t.Fatalf("unexpected pagination: %+v", resp.Pagination)
	}
}
