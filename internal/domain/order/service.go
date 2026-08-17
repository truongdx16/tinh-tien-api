package order

import (
	"errors"
	"time"

	"tinh-tien-api/internal/domain/inventory"
	"tinh-tien-api/internal/domain/product"
	"gorm.io/gorm"
)

var (
	ErrNotFound            = errors.New("order not found")
	ErrInvalidStatus       = errors.New("invalid status transition")
	ErrEmptyItems          = errors.New("order must have at least one item")
	ErrPaymentExceedsTotal = errors.New("payment exceeds balance due")
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(o *Order) error {
	return r.db.Create(o).Error
}

func (r *Repository) Save(o *Order) error {
	return r.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(o).Error
}

func (r *Repository) Get(id string) (*Order, error) {
	var o Order
	err := r.db.Preload("Customer").Preload("Items").Preload("Payments").First(&o, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &o, nil
}

func (r *Repository) List(q ListQuery) ([]Order, int64, error) {
	query := r.db.Model(&Order{})
	if q.Status != "" {
		query = query.Where("status = ?", q.Status)
	}
	if q.CustomerID != "" {
		query = query.Where("customer_id = ?", q.CustomerID)
	}
	if q.From != nil && *q.From != "" {
		query = query.Where("created_at >= ?", *q.From)
	}
	if q.To != nil && *q.To != "" {
		query = query.Where("created_at <= ?", *q.To)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Order
	offset := (q.Page - 1) * q.PageSize
	limit := q.PageSize
	if limit <= 0 {
		limit = 20
	}
	err := query.Preload("Customer").Preload("Items").Order("created_at desc").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func (r *Repository) ListReceivables(page, pageSize int) ([]ReceivableItem, int64, error) {
	base := r.db.Model(&Order{}).
		Where("balance_due > 0 AND status NOT IN ?", []Status{StatusCancelled})
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var orders []Order
	offset := (page - 1) * pageSize
	err := r.db.Preload("Customer").
		Where("balance_due > 0 AND status NOT IN ?", []Status{StatusCancelled}).
		Order("created_at desc").
		Offset(offset).Limit(pageSize).
		Find(&orders).Error
	if err != nil {
		return nil, 0, err
	}
	items := make([]ReceivableItem, 0, len(orders))
	for _, o := range orders {
		name := ""
		cid := ""
		if o.Customer != nil {
			name = o.Customer.Name
			cid = o.Customer.ID.String()
		} else if o.CustomerID != nil {
			cid = *o.CustomerID
		}
		items = append(items, ReceivableItem{
			CustomerID:   cid,
			CustomerName: name,
			OrderID:      o.ID.String(),
			BalanceDue:   o.BalanceDue,
			Status:       o.Status,
		})
	}
	return items, total, nil
}

func (r *Repository) WithTx(fn func(tx *gorm.DB) error) error {
	return r.db.Transaction(fn)
}

type Service struct {
	repo      *Repository
	productRepo *product.Repository
	invSvc    *inventory.Service
}

func NewService(repo *Repository, productRepo *product.Repository, invSvc *inventory.Service) *Service {
	return &Service{repo: repo, productRepo: productRepo, invSvc: invSvc}
}

func (s *Service) Create(req CreateOrderRequest, userID string) (*Order, error) {
	if len(req.Items) == 0 {
		return nil, ErrEmptyItems
	}

	items := make([]OrderItem, 0, len(req.Items))
	var subtotal int64
	for _, it := range req.Items {
		p, err := s.productRepo.GetProduct(it.ProductID)
		if err != nil {
			return nil, err
		}
		price := p.SellPrice
		if it.UnitPrice != nil {
			price = *it.UnitPrice
		}
		lineTotal := int64(float64(price) * it.Quantity)
		subtotal += lineTotal
		items = append(items, OrderItem{
			ProductID: it.ProductID,
			Name:      p.Name,
			Unit:      p.Unit,
			Quantity:  it.Quantity,
			UnitPrice: price,
			LineTotal: lineTotal,
		})
	}

	status := req.Status
	if status == "" {
		status = StatusDraft
	}
	ft := req.FulfillmentType
	if ft == "" {
		ft = FulfillmentPickup
	}

	order := &Order{
		CustomerID:      req.CustomerID,
		Status:          status,
		FulfillmentType: ft,
		DeliveryAddress: req.DeliveryAddress,
		Note:            req.Note,
		AllowBackorder:  req.AllowBackorder,
		Subtotal:        subtotal,
		Total:           subtotal,
		BalanceDue:      subtotal,
		CreatedBy:       userID,
		Items:           items,
	}

	err := s.repo.WithTx(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		if status == StatusConfirmed {
			return s.deductStock(order)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.repo.Get(order.ID.String())
}

func (s *Service) UpdateStatus(id string, req UpdateStatusRequest) (*Order, error) {
	order, err := s.repo.Get(id)
	if err != nil {
		return nil, err
	}
	if !canTransition(order.Status, req.Status) {
		return nil, ErrInvalidStatus
	}

	prev := order.Status
	order.Status = req.Status

	err = s.repo.WithTx(func(tx *gorm.DB) error {
		if prev != StatusConfirmed && req.Status == StatusConfirmed {
			if err := s.deductStock(order); err != nil {
				return err
			}
		}
		if req.Status == StatusCancelled && prev != StatusCancelled {
			if prev == StatusConfirmed || prev == StatusPacked || prev == StatusDelivered {
				if err := s.restock(order); err != nil {
					return err
				}
			}
		}
		return tx.Save(order).Error
	})
	if err != nil {
		return nil, err
	}
	return s.repo.Get(id)
}

func (s *Service) AddPayment(id string, req CreatePaymentRequest, userID string) (*Order, error) {
	order, err := s.repo.Get(id)
	if err != nil {
		return nil, err
	}
	if order.Status == StatusCancelled {
		return nil, ErrInvalidStatus
	}
	if req.Amount <= 0 {
		return nil, errors.New("amount must be positive")
	}
	if req.Amount > order.BalanceDue {
		return nil, ErrPaymentExceedsTotal
	}
	method := req.Method
	if method == "" {
		method = PaymentCash
	}
	payment := Payment{
		OrderID:   id,
		Amount:    req.Amount,
		Method:    method,
		Note:      req.Note,
		PaidAt:    time.Now(),
		CreatedBy: userID,
	}
	order.PaidAmount += req.Amount
	order.BalanceDue -= req.Amount

	err = s.repo.WithTx(func(tx *gorm.DB) error {
		if err := tx.Create(&payment).Error; err != nil {
			return err
		}
		return tx.Save(order).Error
	})
	if err != nil {
		return nil, err
	}
	return s.repo.Get(id)
}

func (s *Service) Get(id string) (*Order, error) {
	return s.repo.Get(id)
}

func (s *Service) List(q ListQuery) ([]Order, int64, error) {
	return s.repo.List(q)
}

func (s *Service) ListReceivables(page, pageSize int) ([]ReceivableItem, int64, error) {
	return s.repo.ListReceivables(page, pageSize)
}

func (s *Service) deductStock(order *Order) error {
	for _, item := range order.Items {
		if !order.AllowBackorder {
			if err := s.invSvc.CheckStock(item.ProductID, item.Quantity); err != nil {
				return err
			}
		}
		ref := order.ID.String()
		m := &inventory.Movement{
			ProductID:     item.ProductID,
			Type:          inventory.MovementSale,
			Quantity:      item.Quantity,
			ReferenceID:   &ref,
			Note:          "order sale",
			CreatedBy:     order.CreatedBy,
			AllowNegative: order.AllowBackorder,
		}
		if err := s.invSvc.RecordMovement(m); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) restock(order *Order) error {
	for _, item := range order.Items {
		ref := order.ID.String()
		m := &inventory.Movement{
			ProductID:   item.ProductID,
			Type:        inventory.MovementReturn,
			Quantity:    item.Quantity,
			ReferenceID: &ref,
			Note:        "order cancelled restock",
			CreatedBy:   order.CreatedBy,
		}
		if err := s.invSvc.RecordMovement(m); err != nil {
			return err
		}
	}
	return nil
}

func canTransition(from, to Status) bool {
	if from == to {
		return true
	}
	switch from {
	case StatusDraft:
		return to == StatusConfirmed || to == StatusCancelled
	case StatusConfirmed:
		return to == StatusPacked || to == StatusCancelled
	case StatusPacked:
		return to == StatusDelivered || to == StatusCancelled
	case StatusDelivered, StatusCancelled:
		return false
	default:
		return false
	}
}
