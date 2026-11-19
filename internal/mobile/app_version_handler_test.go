package mobile

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestAppVersionLatest(t *testing.T) {
	dir := t.TempDir()
	h := NewAppVersionMobileHandler(dir, "https://api.example.com/")
	get := func() (*httptest.ResponseRecorder, map[string]any) {
		w := httptest.NewRecorder()
		h.Latest(w, httptest.NewRequest(http.MethodGet, "/app/version", nil))
		var body map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		return w, body
	}
	write := func(name, content string) {
		if err := os.MkdirAll(filepath.Join(dir, "apk"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "apk", name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if w, _ := get(); w.Code != http.StatusNotFound {
		t.Fatalf("no release: %d, want 404", w.Code)
	}

	write("latest.json", `{"version":"2.0.4","build":6,"file":"tinh_tien-2.0.4+6.apk","notes":"Sửa lỗi","min_build":3,"size":10,"sha256":"ABCD"}`)
	if w, _ := get(); w.Code != http.StatusNotFound {
		t.Fatalf("manifest without the apk: %d, want 404", w.Code)
	}

	write("tinh_tien-2.0.4+6.apk", "0123456789")
	w, body := get()
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	data := body["data"].(map[string]any)
	if data["url"] != "https://api.example.com/uploads/apk/tinh_tien-2.0.4+6.apk" {
		t.Errorf("url: %v", data["url"])
	}
	if data["build"].(float64) != 6 || data["min_build"].(float64) != 3 || data["sha256"] != "abcd" {
		t.Errorf("data: %v", data)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Error("answer may be cached")
	}

	// A path in the file name is refused.
	write("latest.json", `{"version":"x","build":7,"file":"../../etc/passwd.apk"}`)
	if w, _ := get(); w.Code != http.StatusInternalServerError {
		t.Fatalf("path traversal: %d, want 500", w.Code)
	}
}
