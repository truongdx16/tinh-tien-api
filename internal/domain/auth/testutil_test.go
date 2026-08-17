package auth_test

import (
	"testing"

	"tinh-tien-api/internal/domain/auth"
	"tinh-tien-api/internal/domain/customer"
	"tinh-tien-api/internal/domain/expense"
	"tinh-tien-api/internal/domain/inventory"
	"tinh-tien-api/internal/domain/order"
	"tinh-tien-api/internal/domain/product"
	"tinh-tien-api/internal/domain/settings"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&auth.User{},
		&customer.Customer{},
		&product.Category{},
		&product.Product{},
		&inventory.Balance{},
		&inventory.Movement{},
		&order.Order{},
		&order.OrderItem{},
		&order.Payment{},
		&expense.Expense{},
		&settings.Setting{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}
