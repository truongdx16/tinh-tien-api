package report

import (
	"net/http"
	"strconv"

	"tinh-tien-api/internal/pkg/httputil"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Sales(w http.ResponseWriter, r *http.Request) {
	q := SalesReportQuery{
		From:    r.URL.Query().Get("from"),
		To:      r.URL.Query().Get("to"),
		GroupBy: r.URL.Query().Get("group_by"),
	}
	if q.From == "" && q.To == "" {
		q.From, q.To = DefaultPeriod()
	}
	items, err := h.svc.SalesReport(q)
	if err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to generate sales report", err.Error())
		return
	}
	httputil.OK(w, "sales report generated", items)
}

func (h *Handler) Products(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" && to == "" {
		from, to = DefaultPeriod()
	}
	items, err := h.svc.ProductStats(from, to)
	if err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to generate product report", err.Error())
		return
	}
	httputil.OK(w, "product report generated", items)
}

func (h *Handler) Crops(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" && to == "" {
		from, to = DefaultPeriod()
	}
	items, err := h.svc.CropStats(from, to)
	if err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to generate crop report", err.Error())
		return
	}
	httputil.OK(w, "crop report generated", items)
}

func (h *Handler) Receivables(w http.ResponseWriter, r *http.Request) {
	page := httputil.ParsePageParams(r)
	items, total, err := h.svc.Receivables(page.Page, page.PageSize)
	if err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to generate receivables report", err.Error())
		return
	}
	httputil.OKWithPagination(w, "receivables report generated", items, httputil.NewPagination(page.Page, page.PageSize, total))
}

func (h *Handler) LowStock(w http.ResponseWriter, r *http.Request) {
	page := httputil.ParsePageParams(r)
	threshold := 5.0
	if v := r.URL.Query().Get("threshold"); v != "" {
		if t, err := strconv.ParseFloat(v, 64); err == nil {
			threshold = t
		}
	}
	items, total, err := h.svc.LowStock(threshold, page.Page, page.PageSize)
	if err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to generate low stock report", err.Error())
		return
	}
	httputil.OKWithPagination(w, "low stock report generated", items, httputil.NewPagination(page.Page, page.PageSize, total))
}

func (h *Handler) Profit(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" && to == "" {
		from, to = DefaultPeriod()
	}
	item, err := h.svc.ProfitSummary(from, to)
	if err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to generate profit summary", err.Error())
		return
	}
	httputil.OK(w, "profit summary generated", item)
}
