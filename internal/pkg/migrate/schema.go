package migrate

import (
	"fmt"

	"gorm.io/gorm"
)

func ColumnExists(db *gorm.DB, table, column string) (bool, error) {
	var count int64
	err := db.Raw(`
		SELECT COUNT(*) FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?
	`, table, column).Scan(&count).Error
	return count > 0, err
}

func IndexExists(db *gorm.DB, table, index string) (bool, error) {
	var count int64
	err := db.Raw(`
		SELECT COUNT(*) FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND INDEX_NAME = ?
	`, table, index).Scan(&count).Error
	return count > 0, err
}

func TableExists(db *gorm.DB, table string) (bool, error) {
	var count int64
	err := db.Raw(`
		SELECT COUNT(*) FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
	`, table).Scan(&count).Error
	return count > 0, err
}

func DropColumnIfExists(db *gorm.DB, table, column string) error {
	exists, err := ColumnExists(db, table, column)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	return db.Exec(fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN `%s`", table, column)).Error
}

func DropIndexIfExists(db *gorm.DB, table, index string) error {
	exists, err := IndexExists(db, table, index)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	return db.Exec(fmt.Sprintf("ALTER TABLE `%s` DROP INDEX `%s`", table, index)).Error
}

func Exec(db *gorm.DB, sql string, args ...any) error {
	return db.Exec(sql, args...).Error
}
