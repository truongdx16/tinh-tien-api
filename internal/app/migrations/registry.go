package migrations

import (
	"tinh-tien-api/internal/domain/auth"
	"tinh-tien-api/internal/domain/crop"
	"tinh-tien-api/internal/domain/customer"
	"tinh-tien-api/internal/domain/expense"
	"tinh-tien-api/internal/domain/feedback"
	"tinh-tien-api/internal/domain/inventory"
	"tinh-tien-api/internal/domain/media"
	"tinh-tien-api/internal/domain/order"
	"tinh-tien-api/internal/domain/planting"
	"tinh-tien-api/internal/domain/product"
	"tinh-tien-api/internal/domain/settings"
	"tinh-tien-api/internal/pkg/migrate"
	"gorm.io/gorm"
)

// All returns ordered migrations. Append new entries when changing the database.
//
// 001 = fresh install (CREATE tables via AutoMigrate).
// 002+ = incremental upgrades (ADD/ALTER columns, data fixes). Do not use RENAME in 001.
func All() []migrate.Migration {
	return []migrate.Migration{
		{
			ID:   "001_initial_schema",
			Name: "create all base tables",
			Up:   up001InitialSchema,
		},
		{
			ID:   "002_mobile_compat",
			Name: "add mobile-compatible columns and tables",
			Up:   up002MobileCompat,
		},
	}
}

func up001InitialSchema(db *gorm.DB) error {
	return db.AutoMigrate(
		&auth.User{},
		&customer.Customer{},
		&product.Category{},
		&product.Product{},
		&inventory.Balance{},
		&inventory.Movement{},
		&order.Order{},
		&order.OrderItem{},
		&order.Payment{},
		&crop.Plot{},
		&crop.CropBatch{},
		&crop.CropActivity{},
		&crop.Harvest{},
		&expense.Expense{},
		&settings.Setting{},
	)
}

func up002MobileCompat(db *gorm.DB) error {
	return db.AutoMigrate(
		// Existing tables — AutoMigrate adds new columns if not present
		&auth.User{},
		&customer.Customer{},   // adds: code, is_walk_in
		&product.Category{},    // adds: active
		&product.Unit{},        // new table
		&product.Product{},     // adds: unit_id, image_url, sensitivity
		&product.ProductCategory{}, // many2many pivot
		&order.Order{},         // adds: discount
		// New tables
		&planting.PlantingSchedule{},
		&planting.SprayTask{},
		&planting.FertilizeTask{},
		&planting.PlantingTask{},
		&media.Media{},
		&feedback.Feedback{},
	)
}
