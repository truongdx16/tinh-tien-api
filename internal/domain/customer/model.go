package customer

import (
	"tinh-tien-api/internal/pkg/model"
)

type Customer struct {
	model.Base
	Code      string `gorm:"size:32;index" json:"code"`
	Name      string `gorm:"size:128;not null" json:"name"`
	Phone     string `gorm:"size:32;index" json:"phone"`
	Address   string `gorm:"size:512" json:"address"`
	Notes     string `gorm:"size:1024" json:"notes"`
	Active    bool   `gorm:"not null;default:true" json:"active"`
	IsWalkIn  bool   `gorm:"not null;default:false" json:"is_walk_in"`
}
