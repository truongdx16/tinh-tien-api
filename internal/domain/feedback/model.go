package feedback

import "tinh-tien-api/internal/pkg/model"

type Feedback struct {
	model.Base
	Content  string `gorm:"size:2048;not null" json:"content"`
	Rating   *int   `gorm:"" json:"rating"`
	FullName string `gorm:"size:128" json:"full_name"`
	UserID   string `gorm:"size:36;index" json:"user_id"`
}
