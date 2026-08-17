package inventory

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound       = errors.New("inventory not found")
	ErrInsufficient   = errors.New("insufficient stock")
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetBalance(productID string) (*Balance, error) {
	var b Balance
	err := r.db.Preload("Product").First(&b, "product_id = ?", productID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &b, nil
}

func (r *Repository) ListBalances(page, pageSize int) ([]Balance, int64, error) {
	var total int64
	if err := r.db.Model(&Balance{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Balance
	offset := (page - 1) * pageSize
	err := r.db.Preload("Product").Order("created_at desc").Offset(offset).Limit(pageSize).Find(&items).Error
	return items, total, err
}

func (r *Repository) ListMovements(productID string, page, pageSize int) ([]Movement, int64, error) {
	query := r.db.Model(&Movement{})
	if productID != "" {
		query = query.Where("product_id = ?", productID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Movement
	offset := (page - 1) * pageSize
	err := query.Preload("Product").Order("created_at desc").Offset(offset).Limit(pageSize).Find(&items).Error
	return items, total, err
}

func (r *Repository) ApplyMovement(tx *gorm.DB, m *Movement) error {
	if err := tx.Create(m).Error; err != nil {
		return err
	}

	var balance Balance
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("product_id = ?", m.ProductID).
		First(&balance).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		balance = Balance{ProductID: m.ProductID, Quantity: 0}
		if err := tx.Create(&balance).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	delta := m.Quantity
	switch m.Type {
	case MovementSale, MovementWaste:
		delta = -m.Quantity
	case MovementReturn, MovementHarvest, MovementAdjust:
		delta = m.Quantity
	}

	newQty := balance.Quantity + delta
	if newQty < 0 && !m.AllowNegative {
		return fmt.Errorf("%w: product %s has %.2f, need %.2f", ErrInsufficient, m.ProductID, balance.Quantity, m.Quantity)
	}
	return tx.Model(&balance).Update("quantity", newQty).Error
}

func (r *Repository) WithTx(fn func(tx *gorm.DB) error) error {
	return r.db.Transaction(fn)
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListBalances(q ListQuery) ([]BalanceResponse, int64, error) {
	items, total, err := s.repo.ListBalances(q.Page, q.PageSize)
	if err != nil {
		return nil, 0, err
	}
	resp := make([]BalanceResponse, 0, len(items))
	for _, b := range items {
		lowStock := false
		if q.LowStockThreshold > 0 && b.Quantity <= q.LowStockThreshold {
			lowStock = true
		}
		name := ""
		unit := ""
		if b.Product.ID.String() != "" {
			name = b.Product.Name
			unit = b.Product.Unit
		}
		resp = append(resp, BalanceResponse{
			ProductID:   b.ProductID,
			ProductName: name,
			Unit:        unit,
			Quantity:    b.Quantity,
			LowStock:    lowStock,
		})
	}
	return resp, total, nil
}

func (s *Service) ListMovements(productID string, page, pageSize int) ([]Movement, int64, error) {
	return s.repo.ListMovements(productID, page, pageSize)
}

func (s *Service) Adjust(req AdjustmentRequest, userID string) (*Movement, error) {
	if req.ProductID == "" || req.Quantity == 0 {
		return nil, errors.New("product_id and quantity are required")
	}
	m := &Movement{
		ProductID: req.ProductID,
		Type:      MovementAdjust,
		Quantity:  req.Quantity,
		Note:      req.Note,
		CreatedBy: userID,
	}
	err := s.repo.WithTx(func(tx *gorm.DB) error {
		return s.repo.ApplyMovement(tx, m)
	})
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Service) RecordMovement(m *Movement) error {
	return s.repo.WithTx(func(tx *gorm.DB) error {
		return s.repo.ApplyMovement(tx, m)
	})
}

func (s *Service) GetBalance(productID string) (*Balance, error) {
	return s.repo.GetBalance(productID)
}

func (s *Service) CheckStock(productID string, qty float64) error {
	b, err := s.repo.GetBalance(productID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrInsufficient
		}
		return err
	}
	if b.Quantity < qty {
		return ErrInsufficient
	}
	return nil
}
