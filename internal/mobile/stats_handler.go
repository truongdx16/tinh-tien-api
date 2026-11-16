package mobile

import (
	"net/http"
	"strconv"
	"time"

	"tinh-tien-api/internal/domain/customer"
	"tinh-tien-api/internal/domain/report"
	"tinh-tien-api/internal/mobile/adapter"
)

// MobileRevenueDailyDto maps to Flutter DailyRevenue.
type MobileRevenueDailyDto struct {
	SummaryDate string `json:"summary_date"`
	OrderCount  int64  `json:"order_count"`
	Subtotal    string `json:"subtotal"`
	Discount    string `json:"discount"`
	Revenue     string `json:"revenue"`
}

// MobileRevenueTotalsDto maps to Flutter RevenueTotals.
type MobileRevenueTotalsDto struct {
	OrderCount int64  `json:"order_count"`
	Subtotal   string `json:"subtotal"`
	Discount   string `json:"discount"`
	Revenue    string `json:"revenue"`
}

// MobileRevenueStatsDto is the Flutter RevenueReport.
type MobileRevenueStatsDto struct {
	Daily  []MobileRevenueDailyDto `json:"daily"`
	Totals MobileRevenueTotalsDto  `json:"totals"`
}

// MobileTopProductDto maps to Flutter TopProduct.
type MobileTopProductDto struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	SalesQuantity string `json:"sales_quantity"`
	Revenue       string `json:"revenue"`
	Price         string `json:"price"`
}

// MobileCustomerReportDto maps to Flutter CustomerReport.
type MobileCustomerReportDto struct {
	CustomerID   *string `json:"customer_id"`
	CustomerName string  `json:"customer_name"`
	OrderCount   int64   `json:"order_count"`
	Revenue      string  `json:"revenue"`
	Debt         string  `json:"debt"`
}

type StatsMobileHandler struct {
	svc *report.Service
	loc *time.Location
}

func NewStatsMobileHandler(svc *report.Service, loc *time.Location) *StatsMobileHandler {
	return &StatsMobileHandler{svc: svc, loc: loc}
}

func (h *StatsMobileHandler) Revenue(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, to, err := adapter.DayRange(q.Get("from"), q.Get("to"), h.loc)
	if err != nil {
		adapter.Fail(w, http.StatusBadRequest, userMessage(err, "Ngày không hợp lệ"))
		return
	}
	rows, err := h.svc.DailyRevenue(from, to, adapter.TZOffset(h.loc, time.Now()))
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "Không tải được thống kê doanh thu")
		return
	}

	daily := make([]MobileRevenueDailyDto, 0, len(rows))
	var totals report.DailyRevenue
	for _, d := range rows {
		daily = append(daily, MobileRevenueDailyDto{
			SummaryDate: d.SummaryDate,
			OrderCount:  d.OrderCount,
			Subtotal:    adapter.MoneyStr(d.Subtotal),
			Discount:    adapter.MoneyStr(d.Discount),
			Revenue:     adapter.MoneyStr(d.Revenue),
		})
		totals.OrderCount += d.OrderCount
		totals.Subtotal += d.Subtotal
		totals.Discount += d.Discount
		totals.Revenue += d.Revenue
	}
	adapter.OK(w, "revenue stats retrieved", MobileRevenueStatsDto{
		Daily: daily,
		Totals: MobileRevenueTotalsDto{
			OrderCount: totals.OrderCount,
			Subtotal:   adapter.MoneyStr(totals.Subtotal),
			Discount:   adapter.MoneyStr(totals.Discount),
			Revenue:    adapter.MoneyStr(totals.Revenue),
		},
	})
}

func (h *StatsMobileHandler) TopProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 10
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	// Flutter sends only `limit` (all-time); from/to are optional extensions.
	from, to, err := adapter.DayRange(q.Get("from"), q.Get("to"), h.loc)
	if err != nil {
		adapter.Fail(w, http.StatusBadRequest, userMessage(err, "Ngày không hợp lệ"))
		return
	}
	rows, err := h.svc.TopProducts(from, to, limit)
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "Không tải được thống kê sản phẩm")
		return
	}
	dtos := make([]MobileTopProductDto, 0, len(rows))
	for _, p := range rows {
		dtos = append(dtos, MobileTopProductDto{
			ID:            p.ProductID,
			Name:          p.Name,
			SalesQuantity: adapter.FloatStr(p.SalesQuantity),
			Revenue:       adapter.MoneyStr(p.Revenue),
			Price:         adapter.MoneyStr(p.Price),
		})
	}
	adapter.OK(w, "product stats retrieved", dtos)
}

func (h *StatsMobileHandler) Customers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, to, err := adapter.DayRange(q.Get("from"), q.Get("to"), h.loc)
	if err != nil {
		adapter.Fail(w, http.StatusBadRequest, userMessage(err, "Ngày không hợp lệ"))
		return
	}
	rows, err := h.svc.CustomerRevenue(from, to)
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "Không tải được thống kê khách hàng")
		return
	}
	dtos := make([]MobileCustomerReportDto, 0, len(rows))
	for _, c := range rows {
		id := c.CustomerID
		name := c.CustomerName
		if c.IsWalkIn && c.Code == customer.WalkInCode {
			walkIn := customer.WalkInCode
			id = &walkIn
		}
		if name == "" {
			name = "Khách vãng lai"
		}
		dtos = append(dtos, MobileCustomerReportDto{
			CustomerID:   id,
			CustomerName: name,
			OrderCount:   c.OrderCount,
			Revenue:      adapter.MoneyStr(c.Revenue),
			Debt:         adapter.MoneyStr(c.Debt),
		})
	}
	adapter.OK(w, "customer stats retrieved", dtos)
}
