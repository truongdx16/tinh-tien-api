package order

import (
	"errors"
	"github.com/google/uuid"
	"math"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"tinh-tien-api/internal/domain/inventory"
	"tinh-tien-api/internal/domain/product"
)

var (
	ErrNotFound            = errors.New("order not found")
	ErrInvalidClientID     = errors.New("order id must be a uuid")
	ErrInvalidCreatedAt    = errors.New("order date is too old or in the future")
	ErrInvalidStatus       = errors.New("invalid status transition")
	ErrEmptyItems          = errors.New("order must have at least one item")
	ErrPaymentExceedsTotal = errors.New("payment exceeds balance due")
	ErrOrderClosed         = errors.New("order is delivered or cancelled")
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
	err := r.db.Preload("Customer", withDeletedCustomer).Preload("Items").Preload("Payments").First(&o, "id = ?", id).Error
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
	if q.FromTime != nil {
		query = query.Where("created_at >= ?", *q.FromTime)
	} else if q.From != nil && *q.From != "" {
		query = query.Where("created_at >= ?", *q.From)
	}
	if q.ToTime != nil {
		query = query.Where("created_at < ?", *q.ToTime)
	} else if q.To != nil && *q.To != "" {
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
	err := query.Preload("Customer", withDeletedCustomer).Preload("Items").Order("created_at desc").Offset(offset).Limit(limit).Find(&items).Error
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
	err := r.db.Preload("Customer", withDeletedCustomer).
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
	repo        *Repository
	productRepo *product.Repository
	invSvc      *inventory.Service
}

func NewService(repo *Repository, productRepo *product.Repository, invSvc *inventory.Service) *Service {
	return &Service{repo: repo, productRepo: productRepo, invSvc: invSvc}
}

// CreateIdempotent creates the order, or returns the one already created
// with the same client-chosen req.ID (existed = true) — so an app retrying a
// sale after a lost response never records it twice.
func (s *Service) CreateIdempotent(req CreateOrderRequest, userID string) (o *Order, existed bool, err error) {
	if req.ID != nil {
		if _, err := uuid.Parse(*req.ID); err != nil {
			return nil, false, ErrInvalidClientID
		}
		if found, err := s.repo.Get(*req.ID); err == nil {
			return found, true, nil
		} else if !errors.Is(err, ErrNotFound) {
			return nil, false, err
		}
	}
	o, err = s.Create(req, userID)
	if err != nil && req.ID != nil {
		// Two retries racing: the other one inserted it first.
		if found, getErr := s.repo.Get(*req.ID); getErr == nil {
			return found, true, nil
		}
	}
	return o, false, err
}

func (s *Service) Create(req CreateOrderRequest, userID string) (*Order, error) {
	if len(req.Items) == 0 {
		return nil, ErrEmptyItems
	}
	if req.CreatedAt != nil {
		now := time.Now()
		if req.CreatedAt.After(now.Add(10*time.Minute)) || req.CreatedAt.Before(now.Add(-MaxBackdate)) {
			return nil, ErrInvalidCreatedAt
		}
	}

	items := make([]OrderItem, 0, len(req.Items))
	var subtotal int64
	for _, it := range req.Items {
		item, err := s.newItem(it)
		if err != nil {
			return nil, err
		}
		subtotal += item.LineTotal
		items = append(items, item)
	}

	status := req.Status
	if status == "" {
		status = StatusDraft
	}
	ft := req.FulfillmentType
	if ft == "" {
		ft = FulfillmentPickup
	}

	discount, total, paid := computeTotals(subtotal, req.Discount, req.PaidAmount)

	order := &Order{
		CustomerID:      req.CustomerID,
		Status:          status,
		FulfillmentType: ft,
		DeliveryAddress: req.DeliveryAddress,
		Note:            req.Note,
		AllowBackorder:  req.AllowBackorder,
		Discount:        discount,
		Subtotal:        subtotal,
		Total:           total,
		PaidAmount:      paid,
		BalanceDue:      total - paid,
		CreatedBy:       userID,
		Items:           items,
	}
	if req.ID != nil {
		id, err := uuid.Parse(*req.ID)
		if err != nil {
			return nil, ErrInvalidClientID
		}
		order.ID = id
	}
	if req.CreatedAt != nil {
		order.CreatedAt = req.CreatedAt.UTC()
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

// newItem builds an order line priced from the product catalog (or req.UnitPrice).
func (s *Service) newItem(req OrderItemRequest) (OrderItem, error) {
	p, err := s.productRepo.GetProduct(req.ProductID)
	if err != nil {
		return OrderItem{}, err
	}
	price := p.SellPrice
	if req.UnitPrice != nil {
		price = *req.UnitPrice
	}
	unit := p.Unit
	if p.UnitRef != nil && p.UnitRef.Name != "" {
		unit = p.UnitRef.Name
	}
	return OrderItem{
		ProductID: req.ProductID,
		Name:      p.Name,
		Unit:      unit,
		Quantity:  req.Quantity,
		UnitPrice: price,
		LineTotal: lineTotal(price, req.Quantity),
	}, nil
}

func lineTotal(price int64, quantity float64) int64 {
	return int64(math.Round(float64(price) * quantity))
}

// computeTotals clamps discount to [0, subtotal] and paid to [0, total].
// A nil paid means the order is paid in full.
func computeTotals(subtotal, discount int64, paid *int64) (appliedDiscount, total, paidAmount int64) {
	if discount < 0 {
		discount = 0
	}
	if discount > subtotal {
		discount = subtotal
	}
	total = subtotal - discount
	paidAmount = total
	if paid != nil {
		paidAmount = *paid
		if paidAmount < 0 {
			paidAmount = 0
		}
		if paidAmount > total {
			paidAmount = total
		}
	}
	return discount, total, paidAmount
}

// Update changes the lines, discount and payment of an order that is still
// open. Lines already on the order keep their original price and name, so an
// edit never reprices a sale. Stock already deducted is returned and deducted
// again for the new quantities.
func (s *Service) Update(id string, req UpdateOrderRequest) (*Order, error) {
	if len(req.Items) == 0 {
		return nil, ErrEmptyItems
	}
	order, err := s.repo.Get(id)
	if err != nil {
		return nil, err
	}
	if order.Status == StatusDelivered || order.Status == StatusCancelled {
		return nil, ErrOrderClosed
	}
	deducted, err := s.invSvc.HasMovement(order.ID.String(), inventory.MovementSale)
	if err != nil {
		return nil, err
	}

	existing := make(map[string]OrderItem, len(order.Items))
	for _, it := range order.Items {
		existing[it.ProductID] = it
	}
	items := make([]OrderItem, 0, len(req.Items))
	var subtotal int64
	for _, it := range req.Items {
		item := OrderItem{}
		if old, ok := existing[it.ProductID]; ok {
			item = OrderItem{
				ProductID: old.ProductID,
				Name:      old.Name,
				Unit:      old.Unit,
				Quantity:  it.Quantity,
				UnitPrice: old.UnitPrice,
				LineTotal: lineTotal(old.UnitPrice, it.Quantity),
			}
		} else if item, err = s.newItem(it); err != nil {
			return nil, err
		}
		item.OrderID = id
		subtotal += item.LineTotal
		items = append(items, item)
	}

	err = s.repo.WithTx(func(tx *gorm.DB) error {
		if deducted {
			if err := s.restock(order); err != nil {
				return err
			}
		}
		if err := tx.Unscoped().Where("order_id = ?", id).Delete(&OrderItem{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&items).Error; err != nil {
			return err
		}
		order.Items = items
		order.Discount, order.Total, order.PaidAmount = computeTotals(subtotal, req.Discount, req.PaidAmount)
		order.Subtotal = subtotal
		order.BalanceDue = order.Total - order.PaidAmount
		if err := tx.Omit(clause.Associations).Save(order).Error; err != nil {
			return err
		}
		if deducted {
			return s.deductStock(order)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.repo.Get(id)
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
			// Only return stock that was actually deducted (POS orders and imported
			// history never created sale movements).
			deducted, err := s.invSvc.HasMovement(order.ID.String(), inventory.MovementSale)
			if err != nil {
				return err
			}
			if deducted {
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

// DeleteOrder removes an order (soft delete, with its items and payments),
// so it leaves history and statistics. Stock the order took out and that was
// not returned yet (the order was not cancelled) goes back first.
func (s *Service) DeleteOrder(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return ErrNotFound
	}
	order, err := s.repo.Get(id)
	if err != nil {
		return err
	}
	return s.repo.WithTx(func(tx *gorm.DB) error {
		if order.Status != StatusCancelled && s.invSvc != nil {
			ref := order.ID.String()
			deducted, err := s.invSvc.HasMovement(ref, inventory.MovementSale)
			if err != nil {
				return err
			}
			returned, err := s.invSvc.HasMovement(ref, inventory.MovementReturn)
			if err != nil {
				return err
			}
			if deducted && !returned {
				if err := s.restock(order); err != nil {
					return err
				}
			}
		}
		if err := tx.Where("order_id = ?", id).Delete(&OrderItem{}).Error; err != nil {
			return err
		}
		if err := tx.Where("order_id = ?", id).Delete(&Payment{}).Error; err != nil {
			return err
		}
		return tx.Delete(&Order{}, "id = ?", id).Error
	})
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
		// The mobile app has no "packed" step: processing goes straight to completed.
		return to == StatusPacked || to == StatusDelivered || to == StatusCancelled
	case StatusPacked:
		return to == StatusDelivered || to == StatusCancelled
	case StatusDelivered:
		// POS sales are completed immediately; the shop must still be able to void them.
		return to == StatusCancelled
	case StatusCancelled:
		return false
	default:
		return false
	}
}

// withDeletedCustomer loads an order's customer even after the customer was
// (soft) deleted, so old orders keep showing who bought.
func withDeletedCustomer(db *gorm.DB) *gorm.DB { return db.Unscoped() }
