package mobile

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"tinh-tien-api/internal/domain/media"
	"tinh-tien-api/internal/mobile/adapter"
)

type MobileMediaDto struct {
	ID       string  `json:"id"`
	FileURL  string  `json:"file_url"`
	FileName *string `json:"file_name"`
	FilePath *string `json:"file_path"`
}

type MediaMobileHandler struct {
	svc     *media.Service
	baseURL string // e.g. http://localhost:2170
	uploadDir string
}

func NewMediaMobileHandler(svc *media.Service, baseURL, uploadDir string) *MediaMobileHandler {
	return &MediaMobileHandler{svc: svc, baseURL: baseURL, uploadDir: uploadDir}
}

func (h *MediaMobileHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List()
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "failed to list media")
		return
	}
	dtos := make([]MobileMediaDto, 0, len(items))
	for _, m := range items {
		dtos = append(dtos, toMediaDto(m))
	}
	adapter.OK(w, "media retrieved", dtos)
}

func (h *MediaMobileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		adapter.Fail(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		adapter.Fail(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	entityType := r.FormValue("entity_type")
	entityID := r.FormValue("entity_id")

	// Ensure upload directory exists
	if err := os.MkdirAll(h.uploadDir, 0755); err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "failed to create upload directory")
		return
	}

	ext := filepath.Ext(header.Filename)
	fileName := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	filePath := filepath.Join(h.uploadDir, fileName)

	dst, err := os.Create(filePath)
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "failed to save file")
		return
	}
	defer dst.Close()

	size, err := io.Copy(dst, file)
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "failed to write file")
		return
	}

	fileURL := fmt.Sprintf("%s/uploads/%s", h.baseURL, fileName)
	relPath := fmt.Sprintf("uploads/%s", fileName)

	m := &media.Media{
		FileURL:    fileURL,
		FileName:   header.Filename,
		FilePath:   relPath,
		FileSize:   size,
		MimeType:   header.Header.Get("Content-Type"),
		EntityType: entityType,
		EntityID:   entityID,
	}
	saved, err := h.svc.Save(m)
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "failed to save media record")
		return
	}
	adapter.Created(w, "media uploaded", toMediaDto(*saved))
}

func toMediaDto(m media.Media) MobileMediaDto {
	fn := m.FileName
	fp := m.FilePath
	return MobileMediaDto{
		ID:       m.ID.String(),
		FileURL:  m.FileURL,
		FileName: &fn,
		FilePath: &fp,
	}
}
