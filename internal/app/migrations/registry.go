package migrations

import (
	"gorm.io/gorm"
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
		{
			ID:   "003_production_fixes",
			Name: "int feedback ids, order date indexes",
			Up:   up003ProductionFixes,
		},
		{
			ID:   "004_uuid_reference_columns",
			Name: "varchar(36) id references → native uuid (16 bytes)",
			Up:   up004UUIDReferenceColumns,
		},
		{
			ID:   "005_user_phone",
			Name: "users.phone (unique, nullable) for phone sign-in",
			Up:   up005UserPhone,
		},
	}
}

// up005UserPhone adds users.phone: a second way to sign in. Nullable and
// unique (MariaDB allows many NULLs under a unique index).
func up005UserPhone(db *gorm.DB) error {
	exists, err := migrate.ColumnExists(db, "users", "phone")
	if err != nil {
		return err
	}
	if !exists {
		if err := migrate.Exec(db, "ALTER TABLE `users` ADD COLUMN `phone` VARCHAR(16) NULL AFTER `username`"); err != nil {
			return err
		}
	}
	hasIndex, err := migrate.IndexExists(db, "users", "idx_users_phone")
	if err != nil {
		return err
	}
	if !hasIndex {
		return migrate.Exec(db, "CREATE UNIQUE INDEX `idx_users_phone` ON `users` (`phone`)")
	}
	return nil
}

// up004UUIDReferenceColumns stores the last two id references kept as text
// (varchar(36): 37 bytes + the same again in their index) as MariaDB's native
// uuid type (16 bytes), like every other id. Empty or non-uuid values become
// NULL first, since the uuid type does not accept them.
func up004UUIDReferenceColumns(db *gorm.DB) error {
	const uuidPattern = "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"
	columns := []struct{ table, column string }{
		{"media", "entity_id"},
		{"feedbacks", "user_id"},
	}
	for _, c := range columns {
		dataType, err := migrate.ColumnType(db, c.table, c.column)
		if err != nil {
			return err
		}
		if dataType == "" || dataType == "uuid" {
			continue // missing (fresh install creates it as uuid) or already done
		}
		if err := migrate.Exec(db, "UPDATE `"+c.table+"` SET `"+c.column+"` = NULL WHERE `"+
			c.column+"` IS NOT NULL AND `"+c.column+"` NOT REGEXP ?", uuidPattern); err != nil {
			return err
		}
		if err := migrate.Exec(db, "ALTER TABLE `"+c.table+"` MODIFY `"+c.column+"` UUID NULL"); err != nil {
			return err
		}
	}
	return nil
}

// up003ProductionFixes:
//   - feedbacks.id uuid → auto-increment int (Flutter FeedbackNote.id is int); rows are copied.
//   - indexes for date-range queries used by /api/v1/orders and /api/v1/stats.
func up003ProductionFixes(db *gorm.DB) error {
	var idType string
	if err := db.Raw(`SELECT DATA_TYPE FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'feedbacks' AND COLUMN_NAME = 'id'`).
		Scan(&idType).Error; err != nil {
		return err
	}
	if idType != "" && idType != "int" && idType != "bigint" {
		if err := migrate.Exec(db, "RENAME TABLE feedbacks TO feedbacks_uuid_old"); err != nil {
			return err
		}
		if err := db.AutoMigrate(&feedback.Feedback{}); err != nil {
			return err
		}
		if err := migrate.Exec(db, `INSERT INTO feedbacks (created_at, updated_at, deleted_at, content, rating, full_name, user_id)
			SELECT created_at, updated_at, deleted_at, content, rating, full_name, user_id
			FROM feedbacks_uuid_old ORDER BY created_at`); err != nil {
			return err
		}
		if err := migrate.Exec(db, "DROP TABLE feedbacks_uuid_old"); err != nil {
			return err
		}
	} else if err := db.AutoMigrate(&feedback.Feedback{}); err != nil {
		return err
	}

	indexes := []struct{ table, name, cols string }{
		{"orders", "idx_orders_created_at", "created_at"},
		{"orders", "idx_orders_status_created_at", "status, created_at"},
		{"customers", "idx_customers_is_walk_in", "is_walk_in"},
	}
	for _, ix := range indexes {
		exists, err := migrate.IndexExists(db, ix.table, ix.name)
		if err != nil {
			return err
		}
		if !exists {
			if err := migrate.Exec(db, "CREATE INDEX "+ix.name+" ON "+ix.table+" ("+ix.cols+")"); err != nil {
				return err
			}
		}
	}
	return nil
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
		&customer.Customer{},       // adds: code, is_walk_in
		&product.Category{},        // adds: active
		&product.Unit{},            // new table
		&product.Product{},         // adds: unit_id, image_url, sensitivity
		&product.ProductCategory{}, // many2many pivot
		&order.Order{},             // adds: discount
		// New tables
		&planting.PlantingSchedule{},
		&planting.SprayTask{},
		&planting.FertilizeTask{},
		&planting.PlantingTask{},
		&media.Media{},
		&feedback.Feedback{},
	)
}
