package product

import (
	"errors"
	"fmt"
	"strings"

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

// ValidationError is a rejected product input; Msg is safe to show users.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

// validateProduct checks what a product must have. Empty name/price are
// only checked when given (nil = unchanged on update).
func validateProduct(name *string, price *int64, categoryIDs []string) error {
	if name != nil && strings.TrimSpace(*name) == "" {
		return &ValidationError{"Tên sản phẩm không được để trống"}
	}
	if name != nil && len([]rune(strings.TrimSpace(*name))) > 128 {
		return &ValidationError{"Tên sản phẩm tối đa 128 ký tự"}
	}
	if price != nil && *price < 0 {
		return &ValidationError{"Giá bán không được âm"}
	}
	for _, id := range categoryIDs {
		if _, err := uuid.Parse(id); err != nil {
			return &ValidationError{"Danh mục không hợp lệ"}
		}
	}
	return nil
}

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
		query = query.Where(
			"products.category_id = ? OR products.id IN (SELECT product_id FROM product_categories WHERE category_id = ?)",
			q.CategoryID, q.CategoryID)
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

// ProductMetrics are aggregates shown by the mobile app on each product.
type ProductMetrics struct {
	Quantity      float64 // stock (balances.quantity)
	SalesQuantity float64 // SUM(order_items.quantity) of non-cancelled orders
	Revenue       int64   // SUM(order_items.line_total) of non-cancelled orders
}

// Metrics returns stock and sales aggregates keyed by product id.
// Table names are used directly to avoid an import cycle with order/inventory.
func (r *Repository) Metrics(productIDs []string) (map[string]ProductMetrics, error) {
	out := make(map[string]ProductMetrics, len(productIDs))
	if len(productIDs) == 0 {
		return out, nil
	}
	type salesRow struct {
		ProductID     string
		SalesQuantity float64
		Revenue       int64
	}
	var sales []salesRow
	err := r.db.Table("order_items").
		Select("order_items.product_id, COALESCE(SUM(order_items.quantity),0) AS sales_quantity, COALESCE(SUM(order_items.line_total),0) AS revenue").
		Joins("JOIN orders ON orders.id = order_items.order_id AND orders.deleted_at IS NULL").
		Where("order_items.deleted_at IS NULL AND orders.status NOT IN ? AND order_items.product_id IN ?",
			[]string{"cancelled", "draft"}, productIDs).
		Group("order_items.product_id").
		Scan(&sales).Error
	if err != nil {
		return nil, err
	}
	for _, s := range sales {
		m := out[s.ProductID]
		m.SalesQuantity, m.Revenue = s.SalesQuantity, s.Revenue
		out[s.ProductID] = m
	}

	type stockRow struct {
		ProductID string
		Quantity  float64
	}
	var stock []stockRow
	if err := r.db.Table("balances").Select("product_id, quantity").
		Where("deleted_at IS NULL AND product_id IN ?", productIDs).
		Scan(&stock).Error; err != nil {
		return nil, err
	}
	for _, s := range stock {
		m := out[s.ProductID]
		m.Quantity = s.Quantity
		out[s.ProductID] = m
	}
	return out, nil
}

type Service struct {
	repo *Repository
}

func (s *Service) Metrics(productIDs []string) (map[string]ProductMetrics, error) {
	return s.repo.Metrics(productIDs)
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
	c := &Category{Name: req.Name, Description: req.Description, Active: true}
	if req.Active != nil {
		c.Active = *req.Active
	}
	if err := s.repo.CreateCategory(c); err != nil {
		return nil, err
	}
	if !c.Active {
		// GORM skips zero-value bool on insert (column default true) → persist explicitly.
		if err := s.repo.UpdateCategory(c); err != nil {
			return nil, err
		}
	}
	return c, nil
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
	if req.Active != nil {
		c.Active = *req.Active
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
	req.Name = strings.TrimSpace(req.Name)
	if err := validateProduct(&req.Name, &req.SellPrice, req.CategoryIDs); err != nil {
		return nil, err
	}
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
		if err := s.repo.SetProductCategories(p.ID.String(), req.CategoryIDs); err != nil {
			return nil, fmt.Errorf("set categories: %w", err)
		}
	}
	return p, nil
}

func (s *Service) UpdateProduct(id string, req UpdateProductRequest) (*Product, error) {
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		req.Name = &trimmed
	}
	if err := validateProduct(req.Name, req.SellPrice, req.CategoryIDs); err != nil {
		return nil, err
	}
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
		if err := s.repo.SetProductCategories(id, req.CategoryIDs); err != nil {
			return nil, fmt.Errorf("set categories: %w", err)
		}
	}
	return p, nil
}

// DeleteProduct soft-deletes a product: it leaves the catalog, while past
// orders keep their own copy of the name and price.
func (s *Service) DeleteProduct(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return ErrNotFound
	}
	if _, err := s.repo.GetProduct(id); err != nil {
		return err
	}
	return s.repo.DeleteProduct(id)
}

func (s *Service) GetProduct(id string) (*Product, error) {
	return s.repo.GetProduct(id)
}

func (s *Service) ListProducts(q ProductListQuery) ([]Product, int64, error) {
	return s.repo.ListProducts(q)
}
