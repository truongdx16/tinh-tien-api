package expense

import (
	"time"

	"tinh-tien-api/internal/pkg/model"
)

type Category string

const (
	CategorySeed       Category = "seed"
	CategoryFertilizer Category = "fertilizer"
	CategoryFuel       Category = "fuel"
	CategoryLabor      Category = "labor"
	CategoryOther      Category = "other"
)

type Expense struct {
	model.Base
	Category    Category  `gorm:"size:32;not null" json:"category"`
	Amount      int64     `gorm:"not null" json:"amount"`
	Description string    `gorm:"size:512;not null" json:"description"`
	ExpenseDate time.Time `gorm:"not null" json:"expense_date"`
	CreatedBy   string    `gorm:"type:uuid" json:"created_by"`
}
