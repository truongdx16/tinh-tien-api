package inventory_test

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

func TestAdjustAndSale(t *testing.T) {
	db := setupTestDB(t)
	productRepo := product.NewRepository(db)
	invRepo := inventory.NewRepository(db)
	invSvc := inventory.NewService(invRepo)

	p := &product.Product{Name: "Rau muống", Unit: "kg", SellPrice: 20000, Active: true}
	if err := productRepo.CreateProduct(p); err != nil {
		t.Fatalf("create product: %v", err)
	}
	pid := p.ID.String()

	_, err := invSvc.Adjust(inventory.AdjustmentRequest{ProductID: pid, Quantity: 10}, "user-1")
	if err != nil {
		t.Fatalf("adjust: %v", err)
	}

	bal, err := invSvc.GetBalance(pid)
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}
	if bal.Quantity != 10 {
		t.Fatalf("expected 10, got %f", bal.Quantity)
	}

	m := &inventory.Movement{
		ProductID: pid, Type: inventory.MovementSale, Quantity: 3,
		CreatedBy: "user-1",
	}
	if err := invSvc.RecordMovement(m); err != nil {
		t.Fatalf("sale: %v", err)
	}

	bal, err = invSvc.GetBalance(pid)
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}
	if bal.Quantity != 7 {
		t.Fatalf("expected 7, got %f", bal.Quantity)
	}
}

func TestInsufficientStock(t *testing.T) {
	db := setupTestDB(t)
	productRepo := product.NewRepository(db)
	invRepo := inventory.NewRepository(db)
	invSvc := inventory.NewService(invRepo)

	p := &product.Product{Name: "Cà chua", Unit: "kg", SellPrice: 30000, Active: true}
	if err := productRepo.CreateProduct(p); err != nil {
		t.Fatalf("create product: %v", err)
	}

	err := invSvc.CheckStock(p.ID.String(), 5)
	if err == nil {
		t.Fatal("expected insufficient stock error")
	}
}
