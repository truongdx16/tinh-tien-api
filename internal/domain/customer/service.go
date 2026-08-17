package customer

import (
	"errors"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("customer not found")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(c *Customer) error {
	return r.db.Create(c).Error
}

func (r *Repository) Update(c *Customer) error {
	return r.db.Save(c).Error
}

func (r *Repository) Delete(id string) error {
	return r.db.Delete(&Customer{}, "id = ?", id).Error
}

func (r *Repository) Get(id string) (*Customer, error) {
	var c Customer
	err := r.db.First(&c, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *Repository) List(q ListQuery) ([]Customer, int64, error) {
	query := r.db.Model(&Customer{})
	if q.Search != "" {
		like := "%" + q.Search + "%"
		query = query.Where("LOWER(name) LIKE LOWER(?) OR LOWER(phone) LIKE LOWER(?)", like, like)
	}
	if q.Active != nil {
		query = query.Where("active = ?", *q.Active)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Customer
	offset := (q.Page - 1) * q.PageSize
	if offset < 0 {
		offset = 0
	}
	limit := q.PageSize
	if limit <= 0 {
		limit = 20
	}
	err := query.Order("name asc").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(req CreateCustomerRequest) (*Customer, error) {
	c := &Customer{
		Name:    req.Name,
		Phone:   req.Phone,
		Address: req.Address,
		Notes:   req.Notes,
		Active:  true,
	}
	return c, s.repo.Create(c)
}

func (s *Service) Update(id string, req UpdateCustomerRequest) (*Customer, error) {
	c, err := s.repo.Get(id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		c.Name = *req.Name
	}
	if req.Phone != nil {
		c.Phone = *req.Phone
	}
	if req.Address != nil {
		c.Address = *req.Address
	}
	if req.Notes != nil {
		c.Notes = *req.Notes
	}
	if req.Active != nil {
		c.Active = *req.Active
	}
	return c, s.repo.Update(c)
}

func (s *Service) Delete(id string) error {
	return s.repo.Delete(id)
}

func (s *Service) Get(id string) (*Customer, error) {
	return s.repo.Get(id)
}

func (s *Service) List(q ListQuery) ([]Customer, int64, error) {
	return s.repo.List(q)
}
