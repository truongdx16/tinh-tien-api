package product

import (
	"tinh-tien-api/internal/pkg/model"
)

type Category struct {
	model.Base
	Name        string `gorm:"size:128;not null;uniqueIndex" json:"name"`
	Description string `gorm:"size:512" json:"description"`
	Active      bool   `gorm:"not null;default:true" json:"active"`
}

// Unit is a unit of measure (kg, bó, túi…).
type Unit struct {
	model.Base
	Name   string `gorm:"size:64;not null;uniqueIndex" json:"name"`
	Slug   string `gorm:"size:64;index" json:"slug"`
	Active bool   `gorm:"not null;default:true" json:"active"`
}

// ProductCategory is the many-to-many pivot between products and categories.
type ProductCategory struct {
	ProductID  string `gorm:"type:uuid;primaryKey" json:"product_id"`
	CategoryID string `gorm:"type:uuid;primaryKey" json:"category_id"`
}

type Product struct {
	model.Base
	CategoryID  *string    `gorm:"type:uuid;index" json:"category_id"` // kept for backward compat
	Category    *Category  `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	UnitID      *string    `gorm:"type:uuid;index" json:"unit_id"`
	UnitRef     *Unit      `gorm:"foreignKey:UnitID" json:"unit_ref,omitempty"`
	Categories  []Category `gorm:"many2many:product_categories;" json:"categories,omitempty"`
	Name        string     `gorm:"size:256;not null" json:"name"`
	Unit        string     `gorm:"size:32;not null" json:"unit"` // legacy string unit
	ImageURL    string     `gorm:"size:512" json:"image_url"`
	Sensitivity string     `gorm:"size:32;default:'0'" json:"sensitivity"`
	SellPrice   int64      `gorm:"not null;default:0" json:"sell_price"`
	CostPrice   int64      `gorm:"not null;default:0" json:"cost_price"`
	Description string     `gorm:"size:512" json:"description"`
	CropType    string     `gorm:"size:128" json:"crop_type"`
	Seasonal    bool       `gorm:"not null;default:false" json:"seasonal"`
	Active      bool       `gorm:"not null;default:true" json:"active"`
}
