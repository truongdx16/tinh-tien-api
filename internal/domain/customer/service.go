package customer

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("customer not found")

// ErrWalkInProtected: the walk-in customer is used by every counter sale and
// cannot be deleted.
var ErrWalkInProtected = errors.New("walk-in customer cannot be deleted")

// WalkInCode is the walk-in customer identifier used by the Flutter client
// (Customer.id defaults to "CUS-0001" in the app).
const WalkInCode = "CUS-0001"

// ErrInvalidMerge: nothing to merge, the target is also a source, or a
// source is the walk-in customer.
var ErrInvalidMerge = errors.New("invalid customer merge")

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

// FindIDByCode returns the UUID of the customer with the given code.
func (r *Repository) FindIDByCode(code string) (string, error) {
	var c Customer
	if err := r.db.Where("code = ?", code).Order("created_at asc").Limit(1).Find(&c).Error; err != nil {
		return "", err
	}
	if c.ID == uuid.Nil {
		return "", ErrNotFound
	}
	return c.ID.String(), nil
}

// Merge moves every order of the source customers to target, fills the
// target's empty phone/address/notes from them, then soft-deletes the
// sources (rows stay in the table). One transaction: all or nothing.
// Returns how many orders were moved.
func (r *Repository) Merge(target string, sources []string) (int64, error) {
	var moved int64
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var t Customer
		if err := tx.First(&t, "id = ?", target).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		var srcs []Customer
		if err := tx.Where("id IN ?", sources).Order("created_at asc").Find(&srcs).Error; err != nil {
			return err
		}
		if len(srcs) != len(sources) {
			return ErrNotFound
		}
		for _, s := range srcs {
			if s.IsWalkIn || s.Code == WalkInCode {
				return ErrInvalidMerge
			}
			if t.Phone == "" {
				t.Phone = s.Phone
			}
			if t.Address == "" {
				t.Address = s.Address
			}
			if t.Notes == "" {
				t.Notes = s.Notes
			}
		}
		res := tx.Table("orders").Where("customer_id IN ?", sources).Update("customer_id", target)
		if res.Error != nil {
			return res.Error
		}
		moved = res.RowsAffected
		if err := tx.Save(&t).Error; err != nil {
			return err
		}
		return tx.Where("id IN ?", sources).Delete(&Customer{}).Error
	})
	return moved, err
}

func (r *Repository) ensureWalkIn(svc *Service) error {
	var c Customer
	if err := r.db.Where("code = ?", "CUS-0001").Limit(1).Find(&c).Error; err != nil {
		return err
	}
	if c.Name != "" {
		return nil
	}
	_, err := svc.Create(CreateCustomerRequest{Code: "CUS-0001", Name: "Khách vãng lai", IsWalkIn: true})
	return err
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
		Code:     req.Code,
		Name:     req.Name,
		Phone:    req.Phone,
		Address:  req.Address,
		Notes:    req.Notes,
		Active:   true,
		IsWalkIn: req.IsWalkIn,
	}
	return c, s.repo.Create(c)
}

func (s *Service) Update(id string, req UpdateCustomerRequest) (*Customer, error) {
	c, err := s.repo.Get(id)
	if err != nil {
		return nil, err
	}
	if req.Code != nil {
		c.Code = *req.Code
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
	if req.IsWalkIn != nil {
		c.IsWalkIn = *req.IsWalkIn
	}
	return c, s.repo.Update(c)
}

// ResolveID accepts either a customer UUID or a customer code (e.g. "CUS-0001")
// and returns the UUID. Flutter sends "CUS-0001" for walk-in sales.
func (s *Service) ResolveID(idOrCode string) (string, error) {
	if _, err := uuid.Parse(idOrCode); err == nil {
		return idOrCode, nil
	}
	return s.repo.FindIDByCode(idOrCode)
}

func (s *Service) EnsureWalkIn() error {
	if err := s.repo.ensureWalkIn(s); err != nil {
		return err
	}
	return nil
}

// Delete soft-deletes a customer (orders keep their customer_id, so history
// and statistics still show the name). The walk-in customer is protected.
func (s *Service) Delete(id string) error {
	c, err := s.repo.Get(id)
	if err != nil {
		return err
	}
	if c.IsWalkIn || c.Code == WalkInCode {
		return ErrWalkInProtected
	}
	return s.repo.Delete(id)
}

func (s *Service) Get(id string) (*Customer, error) {
	return s.repo.Get(id)
}

func (s *Service) List(q ListQuery) ([]Customer, int64, error) {
	return s.repo.List(q)
}

// Merge folds duplicate customers into target: their orders now belong to
// target and the duplicates are soft-deleted. Ids must be UUIDs.
func (s *Service) Merge(target string, sources []string) (int64, error) {
	seen := map[string]bool{target: true}
	var uniq []string
	for _, id := range sources {
		if seen[id] {
			continue
		}
		seen[id] = true
		uniq = append(uniq, id)
	}
	if len(uniq) == 0 {
		return 0, ErrInvalidMerge
	}
	return s.repo.Merge(target, uniq)
}
