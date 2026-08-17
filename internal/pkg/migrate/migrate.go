package migrate

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

type Migration struct {
	ID   string
	Name string
	Up   func(db *gorm.DB) error
}

type record struct {
	ID        string    `gorm:"primaryKey;size:64"`
	Name      string    `gorm:"size:255;not null"`
	AppliedAt time.Time `gorm:"not null"`
}

func (record) TableName() string { return "schema_migrations" }

// Run applies pending migrations in order. Safe to call multiple times.
func Run(db *gorm.DB, migrations []Migration) error {
	if err := db.AutoMigrate(&record{}); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	applied, err := appliedIDs(db)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if applied[m.ID] {
			continue
		}
		log.Printf("migration %s: %s", m.ID, m.Name)
		if err := m.Up(db); err != nil {
			return fmt.Errorf("migration %s (%s): %w", m.ID, m.Name, err)
		}
		if err := db.Create(&record{ID: m.ID, Name: m.Name, AppliedAt: time.Now()}).Error; err != nil {
			return fmt.Errorf("record migration %s: %w", m.ID, err)
		}
		log.Printf("migration %s: done", m.ID)
	}
	return nil
}

func appliedIDs(db *gorm.DB) (map[string]bool, error) {
	var rows []record
	if err := db.Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(rows))
	for _, r := range rows {
		out[r.ID] = true
	}
	return out, nil
}

func Status(db *gorm.DB, migrations []Migration) ([]string, error) {
	if err := db.AutoMigrate(&record{}); err != nil {
		return nil, err
	}
	applied, err := appliedIDs(db)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, m := range migrations {
		state := "pending"
		if applied[m.ID] {
			state = "applied"
		}
		lines = append(lines, fmt.Sprintf("[%s] %s — %s", state, m.ID, m.Name))
	}
	return lines, nil
}
