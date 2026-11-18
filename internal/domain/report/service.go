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
	From    string
	To      string
	GroupBy string // day, week, month
}

type SalesBucket struct {
	Period     string `json:"period"`
	OrderCount int64  `json:"order_count"`
	Revenue    int64  `json:"revenue"`
	PaidAmount int64  `json:"paid_amount"`
	BalanceDue int64  `json:"balance_due"`
}

type ProductStat struct {
	ProductID    string  `json:"product_id"`
	ProductName  string  `json:"product_name"`
	QuantitySold float64 `json:"quantity_sold"`
	Revenue      int64   `json:"revenue"`
}

type CropStat struct {
	CropName     string  `json:"crop_name"`
	HarvestQty   float64 `json:"harvest_qty"`
	SoldQty      float64 `json:"sold_qty"`
	RemainingQty float64 `json:"remaining_qty"`
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
		Select(periodExpr + " as period, COUNT(*) as order_count, COALESCE(SUM(total),0) as revenue, COALESCE(SUM(paid_amount),0) as paid_amount, COALESCE(SUM(balance_due),0) as balance_due").
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
		Joins("JOIN orders ON orders.id = order_items.order_id AND orders.deleted_at IS NULL").
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
		Joins("JOIN orders ON orders.id = order_items.order_id AND orders.deleted_at IS NULL").
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

// ---- Mobile (Flutter) aggregates ----
// from/to are UTC, half-open [from, to). tzOffset like "+07:00" groups days in shop time.

type DailyRevenue struct {
	SummaryDate string
	OrderCount  int64
	Subtotal    int64
	Discount    int64
	Revenue     int64
}

type TopProduct struct {
	ProductID     string
	Name          string
	SalesQuantity float64
	Revenue       int64
	Price         int64
}

type CustomerRevenue struct {
	CustomerID   *string
	CustomerName string
	IsWalkIn     bool
	Code         string
	OrderCount   int64
	Revenue      int64
	Debt         int64
}

var countedStatuses = []order.Status{order.StatusCancelled, order.StatusDraft}

func applyRange(q *gorm.DB, col string, from, to *time.Time) *gorm.DB {
	if from != nil {
		q = q.Where(col+" >= ?", *from)
	}
	if to != nil {
		q = q.Where(col+" < ?", *to)
	}
	return q
}

func (r *Repository) DailyRevenue(from, to *time.Time, tzOffset string) ([]DailyRevenue, error) {
	day := "DATE(CONVERT_TZ(created_at, '+00:00', '" + tzOffset + "'))"
	q := r.db.Model(&order.Order{}).Where("status NOT IN ?", countedStatuses)
	q = applyRange(q, "created_at", from, to)
	var rows []DailyRevenue
	err := q.Select("DATE_FORMAT(" + day + ", '%Y-%m-%d') AS summary_date, COUNT(*) AS order_count, " +
		"COALESCE(SUM(subtotal),0) AS subtotal, COALESCE(SUM(discount),0) AS discount, COALESCE(SUM(total),0) AS revenue").
		Group(day).Order(day + " ASC").Scan(&rows).Error
	return rows, err
}

func (r *Repository) TopProducts(from, to *time.Time, limit int) ([]TopProduct, error) {
	q := r.db.Table("order_items").
		Select("order_items.product_id, COALESCE(MAX(products.name), MAX(order_items.name)) AS name, "+
			"COALESCE(SUM(order_items.quantity),0) AS sales_quantity, COALESCE(SUM(order_items.line_total),0) AS revenue, "+
			"COALESCE(MAX(products.sell_price),0) AS price").
		Joins("JOIN orders ON orders.id = order_items.order_id AND orders.deleted_at IS NULL").
		Joins("LEFT JOIN products ON products.id = order_items.product_id").
		Where("order_items.deleted_at IS NULL AND orders.status NOT IN ?", countedStatuses)
	q = applyRange(q, "orders.created_at", from, to)
	var rows []TopProduct
	err := q.Group("order_items.product_id").Order("revenue DESC").Limit(limit).Scan(&rows).Error
	return rows, err
}

func (r *Repository) CustomerRevenue(from, to *time.Time) ([]CustomerRevenue, error) {
	q := r.db.Table("orders").
		Select("orders.customer_id, COALESCE(MAX(customers.name), '') AS customer_name, "+
			"COALESCE(MAX(customers.is_walk_in), 0) AS is_walk_in, COALESCE(MAX(customers.code), '') AS code, "+
			"COUNT(*) AS order_count, COALESCE(SUM(orders.total),0) AS revenue, COALESCE(SUM(orders.balance_due),0) AS debt").
		Joins("LEFT JOIN customers ON customers.id = orders.customer_id").
		Where("orders.deleted_at IS NULL AND orders.status NOT IN ?", countedStatuses)
	q = applyRange(q, "orders.created_at", from, to)
	var rows []CustomerRevenue
	err := q.Group("orders.customer_id").Order("revenue DESC").Scan(&rows).Error
	return rows, err
}

// CustomerSummary is what one customer bought in a period (cancelled and
// draft orders excluded, like CustomerRevenue).
type CustomerSummary struct {
	OrderCount   int64
	Revenue      int64
	Debt         int64
	FirstOrderAt *time.Time
	LastOrderAt  *time.Time
}

func (r *Repository) CustomerSummary(customerID string, from, to *time.Time) (CustomerSummary, error) {
	base := func() *gorm.DB {
		q := r.db.Model(&order.Order{}).
			Where("customer_id = ? AND status NOT IN ?", customerID, countedStatuses)
		return applyRange(q, "created_at", from, to)
	}

	var sum CustomerSummary
	var totals struct {
		OrderCount int64
		Revenue    int64
		Debt       int64
	}
	err := base().
		Select("COUNT(*) AS order_count, COALESCE(SUM(total),0) AS revenue, COALESCE(SUM(balance_due),0) AS debt").
		Scan(&totals).Error
	if err != nil {
		return sum, err
	}
	sum.OrderCount, sum.Revenue, sum.Debt = totals.OrderCount, totals.Revenue, totals.Debt
	if sum.OrderCount == 0 {
		return sum, nil
	}

	var first, last order.Order
	if err := base().Select("id, created_at").Order("created_at asc").Limit(1).Find(&first).Error; err != nil {
		return sum, err
	}
	if err := base().Select("id, created_at").Order("created_at desc").Limit(1).Find(&last).Error; err != nil {
		return sum, err
	}
	sum.FirstOrderAt, sum.LastOrderAt = &first.CreatedAt, &last.CreatedAt
	return sum, nil
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

func (s *Service) DailyRevenue(from, to *time.Time, tzOffset string) ([]DailyRevenue, error) {
	return s.repo.DailyRevenue(from, to, tzOffset)
}

func (s *Service) TopProducts(from, to *time.Time, limit int) ([]TopProduct, error) {
	return s.repo.TopProducts(from, to, limit)
}

func (s *Service) CustomerSummary(customerID string, from, to *time.Time) (CustomerSummary, error) {
	return s.repo.CustomerSummary(customerID, from, to)
}

func (s *Service) CustomerRevenue(from, to *time.Time) ([]CustomerRevenue, error) {
	return s.repo.CustomerRevenue(from, to)
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
