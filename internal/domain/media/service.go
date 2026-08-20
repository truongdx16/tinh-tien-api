package media

import (
	"errors"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("media not found")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List() ([]Media, error) {
	var items []Media
	err := r.db.Order("created_at desc").Find(&items).Error
	return items, err
}

func (r *Repository) Create(m *Media) error {
	return r.db.Create(m).Error
}

// ---- Service ----

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List() ([]Media, error) {
	return s.repo.List()
}

func (s *Service) Save(m *Media) (*Media, error) {
	if err := s.repo.Create(m); err != nil {
		return nil, err
	}
	return m, nil
}
