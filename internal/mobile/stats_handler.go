package mobile

import (
	"net/http"
	"strconv"

	"tinh-tien-api/internal/domain/report"
	"tinh-tien-api/internal/mobile/adapter"
)

// ensure report package is used
var _ = report.DefaultPeriod

// MobileRevenueDailyDto maps to Flutter RevenueStatsDto daily entry.
type MobileRevenueDailyDto struct {
	SummaryDate string `json:"summary_date"`
	OrderCount  int64  `json:"order_count"`
	Subtotal    string `json:"subtotal"`
	Discount    string `json:"discount"`
	Revenue     string `json:"revenue"`
}

// MobileRevenueTotalsDto maps to Flutter RevenueStatsDto totals.
type MobileRevenueTotalsDto struct {
	OrderCount int64  `json:"order_count"`
	Subtotal   string `json:"subtotal"`
	Discount   string `json:"discount"`
	Revenue    string `json:"revenue"`
}

// MobileRevenueStatsDto is the Flutter RevenueStatsDto.
type MobileRevenueStatsDto struct {
	Daily  []MobileRevenueDailyDto `json:"daily"`
	Totals MobileRevenueTotalsDto  `json:"totals"`
}

// MobileTopProductDto maps to Flutter TopProductStatsDto.
type MobileTopProductDto struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	SalesQuantity string `json:"sales_quantity"`
	Revenue       string `json:"revenue"`
	Price         string `json:"price"`
}

// MobileCustomerReportDto maps to Flutter CustomerReportDto.
type MobileCustomerReportDto struct {
	CustomerID   *string `json:"customer_id"`
	CustomerName string  `json:"customer_name"`
	OrderCount   int64   `json:"order_count"`
	Revenue      string  `json:"revenue"`
	Debt         string  `json:"debt"`
}

type StatsMobileHandler struct {
	svc *report.Service
}

func NewStatsMobileHandler(svc *report.Service) *StatsMobileHandler {
	return &StatsMobileHandler{svc: svc}
}

func (h *StatsMobileHandler) Revenue(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from := q.Get("from")
	to := q.Get("to")

	buckets, err := h.svc.SalesReport(report.SalesReportQuery{From: from, To: to, GroupBy: "day"})
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "failed to get revenue stats")
		return
	}

	daily := make([]MobileRevenueDailyDto, 0, len(buckets))
	var totalOrders int64
	var totalRev int64
	for _, b := range buckets {
		daily = append(daily, MobileRevenueDailyDto{
			SummaryDate: b.Period,
			OrderCount:  b.OrderCount,
			Subtotal:    adapter.MoneyStr(b.Revenue),
			Discount:    "0",
			Revenue:     adapter.MoneyStr(b.Revenue),
		})
		totalOrders += b.OrderCount
		totalRev += b.Revenue
	}

	result := MobileRevenueStatsDto{
		Daily: daily,
		Totals: MobileRevenueTotalsDto{
			OrderCount: totalOrders,
			Subtotal:   adapter.MoneyStr(totalRev),
			Discount:   "0",
			Revenue:    adapter.MoneyStr(totalRev),
		},
	}
	adapter.OK(w, "revenue stats retrieved", result)
}

func (h *StatsMobileHandler) TopProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 10
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	stats, err := h.svc.ProductStats("", "")
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "failed to get product stats")
		return
	}
	if len(stats) > limit {
		stats = stats[:limit]
	}
	dtos := make([]MobileTopProductDto, 0, len(stats))
	for _, s := range stats {
		dtos = append(dtos, MobileTopProductDto{
			ID:            s.ProductID,
			Name:          s.ProductName,
			SalesQuantity: adapter.FloatStr(s.QuantitySold),
			Revenue:       adapter.MoneyStr(s.Revenue),
			Price:         "0",
		})
	}
	adapter.OK(w, "product stats retrieved", dtos)
}

func (h *StatsMobileHandler) Customers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from := q.Get("from")
	to := q.Get("to")
	receivables, _, err := h.svc.Receivables(1, 200)
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "failed to get customer stats")
		return
	}
	_ = from
	_ = to
	dtos := make([]MobileCustomerReportDto, 0, len(receivables))
	for _, rec := range receivables {
		cid := rec.CustomerID
		var cidPtr *string
		if cid != "" {
			cidPtr = &cid
		}
		dtos = append(dtos, MobileCustomerReportDto{
			CustomerID:   cidPtr,
			CustomerName: rec.CustomerName,
			OrderCount:   0,
			Revenue:      "0",
			Debt:         adapter.MoneyStr(rec.BalanceDue),
		})
	}
	adapter.OK(w, "customer stats retrieved", dtos)
}
