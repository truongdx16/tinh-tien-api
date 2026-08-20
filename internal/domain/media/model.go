package media

import "tinh-tien-api/internal/pkg/model"

// Media stores uploaded file references.
type Media struct {
	model.Base
	FileURL    string `gorm:"size:1024;not null" json:"file_url"`
	FileName   string `gorm:"size:256" json:"file_name"`
	FilePath   string `gorm:"size:1024" json:"file_path"`
	FileSize   int64  `gorm:"default:0" json:"file_size"`
	MimeType   string `gorm:"size:128" json:"mime_type"`
	EntityType string `gorm:"size:64;index" json:"entity_type"`
	EntityID   string `gorm:"size:36;index" json:"entity_id"`
}
