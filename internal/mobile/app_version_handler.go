package mobile

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"tinh-tien-api/internal/mobile/adapter"
)

// AppReleaseManifest is <upload_dir>/apk/latest.json, written by the app
// repo's `make release` next to the APK it uploads.
type AppReleaseManifest struct {
	Version    string `json:"version"`     // 2.0.4
	Build      int    `json:"build"`       // 6 (Android versionCode)
	File       string `json:"file"`        // tinh_tien-2.0.4+6.apk, in the same folder
	Notes      string `json:"notes"`       // what changed, shown in the app
	MinBuild   int    `json:"min_build"`   // older builds must update (0 = none)
	Size       int64  `json:"size"`        // bytes
	SHA256     string `json:"sha256"`      // hex, checked by the app after download
	ReleasedAt string `json:"released_at"` // RFC3339
}

// MobileAppVersionDto is the answer of GET /app/version.
type MobileAppVersionDto struct {
	Version    string `json:"version"`
	Build      int    `json:"build"`
	URL        string `json:"url"`
	Notes      string `json:"notes"`
	MinBuild   int    `json:"min_build"`
	Size       int64  `json:"size"`
	SHA256     string `json:"sha256"`
	ReleasedAt string `json:"released_at"`
}

// AppVersionMobileHandler tells the Android app which APK is the latest,
// for in-app updates (the user downloads and installs it themselves).
type AppVersionMobileHandler struct {
	uploadDir     string
	publicBaseURL string
}

func NewAppVersionMobileHandler(uploadDir, publicBaseURL string) *AppVersionMobileHandler {
	return &AppVersionMobileHandler{uploadDir: uploadDir, publicBaseURL: strings.TrimRight(publicBaseURL, "/")}
}

// Latest: GET /app/version (public: the login screen checks too).
func (h *AppVersionMobileHandler) Latest(w http.ResponseWriter, r *http.Request) {
	dir := filepath.Join(h.uploadDir, "apk")
	raw, err := os.ReadFile(filepath.Join(dir, "latest.json"))
	if errors.Is(err, fs.ErrNotExist) {
		adapter.Fail(w, http.StatusNotFound, "Chưa có bản cập nhật")
		return
	}
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "Không đọc được thông tin phiên bản")
		return
	}
	var m AppReleaseManifest
	if err := json.Unmarshal(raw, &m); err != nil || m.Build <= 0 || m.File == "" {
		adapter.Fail(w, http.StatusInternalServerError, "Thông tin phiên bản không hợp lệ")
		return
	}
	// Only a file of that folder: no path in the name.
	name := filepath.Base(m.File)
	if name != m.File || !strings.HasSuffix(strings.ToLower(name), ".apk") {
		adapter.Fail(w, http.StatusInternalServerError, "Thông tin phiên bản không hợp lệ")
		return
	}
	if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
		adapter.Fail(w, http.StatusNotFound, "Chưa có bản cập nhật")
		return
	}
	// Always the latest answer (a new release must show at once).
	w.Header().Set("Cache-Control", "no-store")
	adapter.OK(w, "latest app version", MobileAppVersionDto{
		Version:    m.Version,
		Build:      m.Build,
		URL:        h.publicBaseURL + "/uploads/apk/" + name,
		Notes:      m.Notes,
		MinBuild:   m.MinBuild,
		Size:       m.Size,
		SHA256:     strings.ToLower(m.SHA256),
		ReleasedAt: m.ReleasedAt,
	})
}
