package inventory

import (
	"tinh-tien-api/internal/domain/product"
	"tinh-tien-api/internal/pkg/model"
)

type MovementType string

const (
	MovementHarvest MovementType = "harvest"
	MovementSale    MovementType = "sale"
	MovementAdjust  MovementType = "adjust"
	MovementWaste   MovementType = "waste"
	MovementReturn  MovementType = "return"
)

type Balance struct {
	model.Base
	ProductID string          `gorm:"type:uuid;uniqueIndex;not null" json:"product_id"`
	Product   product.Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Quantity  float64         `gorm:"not null;default:0" json:"quantity"`
}

type Movement struct {
	model.Base
	ProductID   string       `gorm:"type:uuid;index;not null" json:"product_id"`
	Product     product.Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Type        MovementType `gorm:"size:32;not null" json:"type"`
	Quantity    float64      `gorm:"not null" json:"quantity"`
	ReferenceID *string      `gorm:"type:uuid;index" json:"reference_id,omitempty"`
	Note          string       `gorm:"size:512" json:"note"`
	CreatedBy     string       `gorm:"type:uuid" json:"created_by"`
	AllowNegative bool         `gorm:"-" json:"-"`
}
