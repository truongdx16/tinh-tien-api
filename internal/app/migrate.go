package app

import (
	"gorm.io/gorm"
	"tinh-tien-api/internal/app/migrations"
	"tinh-tien-api/internal/pkg/migrate"
)

// Migrate runs versioned database migrations. Safe to call on every startup.
func Migrate(db *gorm.DB) error {
	return migrate.Run(db, migrations.All())
}

// MigrateStatus returns human-readable migration status lines.
func MigrateStatus(db *gorm.DB) ([]string, error) {
	return migrate.Status(db, migrations.All())
}
