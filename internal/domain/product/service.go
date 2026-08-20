package product

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func mustParseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}

var ErrNotFound = errors.New("not found")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ---- Unit repository methods ----

func (r *Repository) CreateUnit(u *Unit) error {
	return r.db.Create(u).Error
}

func (r *Repository) UpdateUnit(u *Unit) error {
	return r.db.Save(u).Error
}

func (r *Repository) GetUnit(id string) (*Unit, error) {
	var u Unit
	err := r.db.First(&u, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *Repository) ListUnits() ([]Unit, error) {
	var items []Unit
	err := r.db.Where("active = ?", true).Order("name asc").Find(&items).Error
	return items, err
}

func (r *Repository) ListAllUnits() ([]Unit, error) {
	var items []Unit
	err := r.db.Order("name asc").Find(&items).Error
	return items, err
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
	err := r.db.Preload("Category").Preload("UnitRef").Preload("Categories").First(&p, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *Repository) SetProductCategories(productID string, categoryIDs []string) error {
	var cats []Category
	for _, cid := range categoryIDs {
		cats = append(cats, Category{})
		cats[len(cats)-1].ID = mustParseUUID(cid)
	}
	var p Product
	p.ID = mustParseUUID(productID)
	return r.db.Model(&p).Association("Categories").Replace(cats)
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
	err := query.Preload("Category").Preload("UnitRef").Preload("Categories").Order("name asc").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ---- Unit service methods ----

func (s *Service) CreateUnit(name, slug string, status int) (*Unit, error) {
	u := &Unit{Name: name, Slug: slug, Active: status == 1}
	return u, s.repo.CreateUnit(u)
}

func (s *Service) UpdateUnit(id string, name *string, slug *string, status *int) (*Unit, error) {
	u, err := s.repo.GetUnit(id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		u.Name = *name
	}
	if slug != nil {
		u.Slug = *slug
	}
	if status != nil {
		u.Active = *status == 1
	}
	return u, s.repo.UpdateUnit(u)
}

func (s *Service) GetUnit(id string) (*Unit, error) {
	return s.repo.GetUnit(id)
}

func (s *Service) ListUnits() ([]Unit, error) {
	return s.repo.ListAllUnits()
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
		UnitID:      req.UnitID,
		Name:        req.Name,
		Unit:        req.Unit,
		ImageURL:    req.ImageURL,
		Sensitivity: req.Sensitivity,
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
	if p.Sensitivity == "" {
		p.Sensitivity = "0"
	}
	if err := s.repo.CreateProduct(p); err != nil {
		return nil, err
	}
	if len(req.CategoryIDs) > 0 {
		_ = s.repo.SetProductCategories(p.ID.String(), req.CategoryIDs)
	}
	return p, nil
}

func (s *Service) UpdateProduct(id string, req UpdateProductRequest) (*Product, error) {
	p, err := s.repo.GetProduct(id)
	if err != nil {
		return nil, err
	}
	if req.CategoryID != nil {
		p.CategoryID = req.CategoryID
	}
	if req.UnitID != nil {
		p.UnitID = req.UnitID
	}
	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Unit != nil {
		p.Unit = *req.Unit
	}
	if req.ImageURL != nil {
		p.ImageURL = *req.ImageURL
	}
	if req.Sensitivity != nil {
		p.Sensitivity = *req.Sensitivity
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
	if err := s.repo.UpdateProduct(p); err != nil {
		return nil, err
	}
	if req.CategoryIDs != nil {
		_ = s.repo.SetProductCategories(id, req.CategoryIDs)
	}
	return p, nil
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
