package inventory

import (
	"errors"
	"net/http"
	"strconv"

	"tinh-tien-api/internal/domain/auth"
	"tinh-tien-api/internal/pkg/httputil"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ListBalances(w http.ResponseWriter, r *http.Request) {
	page := httputil.ParsePageParams(r)
	q := ListQuery{Page: page.Page, PageSize: page.PageSize}
	if v := r.URL.Query().Get("low_stock_threshold"); v != "" {
		threshold, err := strconv.ParseFloat(v, 64)
		if err == nil {
			q.LowStockThreshold = threshold
		}
	}
	items, total, err := h.svc.ListBalances(q)
	if err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to list inventory", err.Error())
		return
	}
	httputil.OKWithPagination(w, "inventory retrieved", items, httputil.NewPagination(page.Page, page.PageSize, total))
}

func (h *Handler) ListMovements(w http.ResponseWriter, r *http.Request) {
	page := httputil.ParsePageParams(r)
	productID := r.URL.Query().Get("product_id")
	items, total, err := h.svc.ListMovements(productID, page.Page, page.PageSize)
	if err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to list movements", err.Error())
		return
	}
	httputil.OKWithPagination(w, "movements retrieved", items, httputil.NewPagination(page.Page, page.PageSize, total))
}

func (h *Handler) Adjust(w http.ResponseWriter, r *http.Request) {
	var req AdjustmentRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	userID := auth.UserIDFromContext(r.Context())
	item, err := h.svc.Adjust(req, userID)
	if err != nil {
		if errors.Is(err, ErrInsufficient) {
			httputil.Fail(w, http.StatusConflict, "insufficient stock", err.Error())
			return
		}
		httputil.Fail(w, http.StatusBadRequest, "failed to adjust inventory", err.Error())
		return
	}
	httputil.Created(w, "inventory adjusted", item)
}

func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	productID := r.URL.Query().Get("product_id")
	if productID == "" {
		httputil.Fail(w, http.StatusBadRequest, "invalid request", "product_id is required")
		return
	}
	item, err := h.svc.GetBalance(productID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.OK(w, "balance retrieved", BalanceResponse{ProductID: productID, Quantity: 0})
			return
		}
		httputil.Fail(w, http.StatusInternalServerError, "failed to get balance", err.Error())
		return
	}
	httputil.OK(w, "balance retrieved", item)
}
