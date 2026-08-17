package report

import (
	"time"

	"gorm.io/gorm"
	"tinh-tien-api/internal/domain/crop"
	"tinh-tien-api/internal/domain/expense"
	"tinh-tien-api/internal/domain/inventory"
	"tinh-tien-api/internal/domain/order"
)

type SalesReportQuery struct {
	From string
	To   string
	GroupBy string // day, week, month
}

type SalesBucket struct {
	Period      string `json:"period"`
	OrderCount  int64  `json:"order_count"`
	Revenue     int64  `json:"revenue"`
	PaidAmount  int64  `json:"paid_amount"`
	BalanceDue  int64  `json:"balance_due"`
}

type ProductStat struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	QuantitySold float64 `json:"quantity_sold"`
	Revenue     int64   `json:"revenue"`
}

type CropStat struct {
	CropName      string  `json:"crop_name"`
	HarvestQty    float64 `json:"harvest_qty"`
	SoldQty       float64 `json:"sold_qty"`
	RemainingQty  float64 `json:"remaining_qty"`
}

type ProfitSummary struct {
	Revenue  int64 `json:"revenue"`
	Expenses int64 `json:"expenses"`
	Profit   int64 `json:"profit"`
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SalesReport(q SalesReportQuery) ([]SalesBucket, error) {
	query := r.db.Model(&order.Order{}).
		Where("status NOT IN ?", []order.Status{order.StatusCancelled, order.StatusDraft})
	if q.From != "" {
		query = query.Where("created_at >= ?", q.From)
	}
	if q.To != "" {
		query = query.Where("created_at <= ?", q.To)
	}

	type row struct {
		Period     string
		OrderCount int64
		Revenue    int64
		PaidAmount int64
		BalanceDue int64
	}

	trunc := "day"
	switch q.GroupBy {
	case "week":
		trunc = "week"
	case "month":
		trunc = "month"
	}
	periodExpr := salesPeriodExpr(trunc)

	var rows []row
	err := query.
		Select(periodExpr+" as period, COUNT(*) as order_count, COALESCE(SUM(total),0) as revenue, COALESCE(SUM(paid_amount),0) as paid_amount, COALESCE(SUM(balance_due),0) as balance_due").
		Group(periodExpr).
		Order(periodExpr + " asc").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	items := make([]SalesBucket, 0, len(rows))
	for _, row := range rows {
		items = append(items, SalesBucket{
			Period: row.Period, OrderCount: row.OrderCount,
			Revenue: row.Revenue, PaidAmount: row.PaidAmount, BalanceDue: row.BalanceDue,
		})
	}
	return items, nil
}

func (r *Repository) ProductStats(from, to string) ([]ProductStat, error) {
	query := r.db.Table("order_items").
		Select("order_items.product_id, order_items.name as product_name, COALESCE(SUM(order_items.quantity),0) as quantity_sold, COALESCE(SUM(order_items.line_total),0) as revenue").
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.status NOT IN ?", []order.Status{order.StatusCancelled, order.StatusDraft})
	if from != "" {
		query = query.Where("orders.created_at >= ?", from)
	}
	if to != "" {
		query = query.Where("orders.created_at <= ?", to)
	}
	var items []ProductStat
	err := query.Group("order_items.product_id, order_items.name").Order("revenue desc").Scan(&items).Error
	return items, err
}

func (r *Repository) CropStats(from, to string) ([]CropStat, error) {
	harvestQuery := r.db.Model(&crop.Harvest{}).Select("product_id, COALESCE(SUM(quantity),0) as qty")
	if from != "" {
		harvestQuery = harvestQuery.Where("harvested_at >= ?", from)
	}
	if to != "" {
		harvestQuery = harvestQuery.Where("harvested_at <= ?", to)
	}
	type harvestRow struct {
		ProductID string
		Qty       float64
	}
	var harvests []harvestRow
	if err := harvestQuery.Group("product_id").Scan(&harvests).Error; err != nil {
		return nil, err
	}

	soldQuery := r.db.Table("order_items").
		Select("order_items.product_id, COALESCE(SUM(order_items.quantity),0) as qty").
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.status NOT IN ?", []order.Status{order.StatusCancelled, order.StatusDraft})
	if from != "" {
		soldQuery = soldQuery.Where("orders.created_at >= ?", from)
	}
	if to != "" {
		soldQuery = soldQuery.Where("orders.created_at <= ?", to)
	}
	type soldRow struct {
		ProductID string
		Qty       float64
	}
	var sold []soldRow
	if err := soldQuery.Group("order_items.product_id").Scan(&sold).Error; err != nil {
		return nil, err
	}

	soldMap := map[string]float64{}
	for _, s := range sold {
		soldMap[s.ProductID] = s.Qty
	}

	items := make([]CropStat, 0, len(harvests))
	for _, h := range harvests {
		var batch crop.CropBatch
		name := h.ProductID
		_ = r.db.Where("product_id = ?", h.ProductID).First(&batch).Error
		if batch.CropName != "" {
			name = batch.CropName
		}
		soldQty := soldMap[h.ProductID]
		items = append(items, CropStat{
			CropName: name, HarvestQty: h.Qty, SoldQty: soldQty, RemainingQty: h.Qty - soldQty,
		})
	}
	return items, nil
}

func (r *Repository) TotalRevenue(from, to string) (int64, error) {
	query := r.db.Model(&order.Order{}).
		Where("status NOT IN ?", []order.Status{order.StatusCancelled, order.StatusDraft})
	if from != "" {
		query = query.Where("created_at >= ?", from)
	}
	if to != "" {
		query = query.Where("created_at <= ?", to)
	}
	var total int64
	err := query.Select("COALESCE(SUM(total),0)").Scan(&total).Error
	return total, err
}

type Service struct {
	repo       *Repository
	expenseSvc *expense.Service
	invSvc     *inventory.Service
	orderSvc   *order.Service
}

func NewService(repo *Repository, expenseSvc *expense.Service, invSvc *inventory.Service, orderSvc *order.Service) *Service {
	return &Service{repo: repo, expenseSvc: expenseSvc, invSvc: invSvc, orderSvc: orderSvc}
}

func (s *Service) SalesReport(q SalesReportQuery) ([]SalesBucket, error) {
	if q.GroupBy == "" {
		q.GroupBy = "day"
	}
	return s.repo.SalesReport(q)
}

func (s *Service) ProductStats(from, to string) ([]ProductStat, error) {
	return s.repo.ProductStats(from, to)
}

func (s *Service) CropStats(from, to string) ([]CropStat, error) {
	return s.repo.CropStats(from, to)
}

func (s *Service) Receivables(page, pageSize int) ([]order.ReceivableItem, int64, error) {
	return s.orderSvc.ListReceivables(page, pageSize)
}

func (s *Service) LowStock(threshold float64, page, pageSize int) ([]inventory.BalanceResponse, int64, error) {
	if threshold <= 0 {
		threshold = 5
	}
	return s.invSvc.ListBalances(inventory.ListQuery{
		LowStockThreshold: threshold,
		Page:              page,
		PageSize:          pageSize,
	})
}

func (s *Service) ProfitSummary(from, to string) (*ProfitSummary, error) {
	revenue, err := s.repo.TotalRevenue(from, to)
	if err != nil {
		return nil, err
	}
	expenses, err := s.expenseSvc.SumByPeriod(from, to)
	if err != nil {
		return nil, err
	}
	return &ProfitSummary{Revenue: revenue, Expenses: expenses, Profit: revenue - expenses}, nil
}

func DefaultPeriod() (string, string) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	return start.Format(time.RFC3339), now.Format(time.RFC3339)
}

func salesPeriodExpr(groupBy string) string {
	switch groupBy {
	case "week":
		return "DATE(DATE_SUB(created_at, INTERVAL WEEKDAY(created_at) DAY))"
	case "month":
		return "DATE_FORMAT(created_at, '%Y-%m-01')"
	default:
		return "DATE(created_at)"
	}
}
