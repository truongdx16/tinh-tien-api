package feedback

import (
	"time"

	"gorm.io/gorm"
)

// Feedback uses an auto-increment integer id: the Flutter FeedbackNote model
// declares `required int id`.
type Feedback struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Content   string         `gorm:"size:2048;not null" json:"content"`
	Rating    *int           `json:"rating"`
	FullName  string         `gorm:"size:128" json:"full_name"`
	UserID    string         `gorm:"size:36;index" json:"user_id"`
}
