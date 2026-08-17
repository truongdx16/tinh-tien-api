package product

import (
	"errors"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("not found")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateCategory(c *Category) error {
	return r.db.Create(c).Error
}

func (r *Repository) UpdateCategory(c *Category) error {
	return r.db.Save(c).Error
}

func (r *Repository) DeleteCategory(id string) error {
	return r.db.Delete(&Category{}, "id = ?", id).Error
}

func (r *Repository) GetCategory(id string) (*Category, error) {
	var c Category
	err := r.db.First(&c, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *Repository) ListCategories(page, pageSize int) ([]Category, int64, error) {
	var total int64
	if err := r.db.Model(&Category{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Category
	offset := (page - 1) * pageSize
	err := r.db.Order("name asc").Offset(offset).Limit(pageSize).Find(&items).Error
	return items, total, err
}

func (r *Repository) CreateProduct(p *Product) error {
	return r.db.Create(p).Error
}

func (r *Repository) UpdateProduct(p *Product) error {
	return r.db.Save(p).Error
}

func (r *Repository) DeleteProduct(id string) error {
	return r.db.Delete(&Product{}, "id = ?", id).Error
}

func (r *Repository) GetProduct(id string) (*Product, error) {
	var p Product
	err := r.db.Preload("Category").First(&p, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *Repository) ListProducts(q ProductListQuery) ([]Product, int64, error) {
	query := r.db.Model(&Product{})
	if q.CategoryID != "" {
		query = query.Where("category_id = ?", q.CategoryID)
	}
	if q.Active != nil {
		query = query.Where("active = ?", *q.Active)
	}
	if q.Search != "" {
		like := "%" + q.Search + "%"
		query = query.Where("LOWER(name) LIKE LOWER(?) OR LOWER(crop_type) LIKE LOWER(?)", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Product
	offset := (q.Page - 1) * q.PageSize
	limit := q.PageSize
	if limit <= 0 {
		limit = 20
	}
	err := query.Preload("Category").Order("name asc").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateCategory(req CreateCategoryRequest) (*Category, error) {
	c := &Category{Name: req.Name, Description: req.Description}
	return c, s.repo.CreateCategory(c)
}

func (s *Service) UpdateCategory(id string, req UpdateCategoryRequest) (*Category, error) {
	c, err := s.repo.GetCategory(id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		c.Name = *req.Name
	}
	if req.Description != nil {
		c.Description = *req.Description
	}
	return c, s.repo.UpdateCategory(c)
}

func (s *Service) DeleteCategory(id string) error {
	return s.repo.DeleteCategory(id)
}

func (s *Service) GetCategory(id string) (*Category, error) {
	return s.repo.GetCategory(id)
}

func (s *Service) ListCategories(page, pageSize int) ([]Category, int64, error) {
	return s.repo.ListCategories(page, pageSize)
}

func (s *Service) CreateProduct(req CreateProductRequest) (*Product, error) {
	p := &Product{
		CategoryID:  req.CategoryID,
		Name:        req.Name,
		Unit:        req.Unit,
		SellPrice:   req.SellPrice,
		CostPrice:   req.CostPrice,
		Description: req.Description,
		CropType:    req.CropType,
		Seasonal:    req.Seasonal,
		Active:      req.Active,
	}
	if p.Unit == "" {
		p.Unit = "kg"
	}
	return p, s.repo.CreateProduct(p)
}

func (s *Service) UpdateProduct(id string, req UpdateProductRequest) (*Product, error) {
	p, err := s.repo.GetProduct(id)
	if err != nil {
		return nil, err
	}
	if req.CategoryID != nil {
		p.CategoryID = req.CategoryID
	}
	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Unit != nil {
		p.Unit = *req.Unit
	}
	if req.SellPrice != nil {
		p.SellPrice = *req.SellPrice
	}
	if req.CostPrice != nil {
		p.CostPrice = *req.CostPrice
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if req.CropType != nil {
		p.CropType = *req.CropType
	}
	if req.Seasonal != nil {
		p.Seasonal = *req.Seasonal
	}
	if req.Active != nil {
		p.Active = *req.Active
	}
	return p, s.repo.UpdateProduct(p)
}

func (s *Service) DeleteProduct(id string) error {
	return s.repo.DeleteProduct(id)
}

func (s *Service) GetProduct(id string) (*Product, error) {
	return s.repo.GetProduct(id)
}

func (s *Service) ListProducts(q ProductListQuery) ([]Product, int64, error) {
	return s.repo.ListProducts(q)
}
