package settings

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Get(key string) (string, error) {
	var s Setting
	err := r.db.Where("setting_key = ?", key).Limit(1).Find(&s).Error
	if err != nil {
		return "", err
	}
	if s.ID == uuid.Nil {
		return "", nil
	}
	return s.Value, nil
}

func (r *Repository) Set(key, value string) error {
	var s Setting
	err := r.db.Where("setting_key = ?", key).Limit(1).Find(&s).Error
	if err != nil {
		return err
	}
	if s.ID == uuid.Nil {
		return r.db.Create(&Setting{Key: key, Value: value}).Error
	}
	s.Value = value
	return r.db.Save(&s).Error
}
