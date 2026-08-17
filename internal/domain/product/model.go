package product

import (
	"tinh-tien-api/internal/pkg/model"
)

type Category struct {
	model.Base
	Name        string `gorm:"size:128;not null;uniqueIndex" json:"name"`
	Description string `gorm:"size:512" json:"description"`
}

type Product struct {
	model.Base
	CategoryID  *string `gorm:"type:uuid;index" json:"category_id"`
	Category    *Category `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Name        string  `gorm:"size:256;not null" json:"name"`
	Unit        string  `gorm:"size:32;not null" json:"unit"`
	SellPrice   int64   `gorm:"not null;default:0" json:"sell_price"`
	CostPrice   int64   `gorm:"not null;default:0" json:"cost_price"`
	Description string  `gorm:"size:512" json:"description"`
	CropType    string  `gorm:"size:128" json:"crop_type"`
	Seasonal    bool    `gorm:"not null;default:false" json:"seasonal"`
	Active      bool    `gorm:"not null;default:true" json:"active"`
}
