package feedback

import (
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List() ([]Feedback, error) {
	var items []Feedback
	err := r.db.Order("created_at desc").Find(&items).Error
	return items, err
}

func (r *Repository) Create(f *Feedback) error {
	return r.db.Create(f).Error
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List() ([]Feedback, error) {
	return s.repo.List()
}

func (s *Service) Create(content string, rating *int, fullName, userID string) (*Feedback, error) {
	f := &Feedback{Content: content, Rating: rating, FullName: fullName, UserID: userID}
	if err := s.repo.Create(f); err != nil {
		return nil, err
	}
	return f, nil
}
