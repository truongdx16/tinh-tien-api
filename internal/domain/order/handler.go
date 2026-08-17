package order

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"tinh-tien-api/internal/domain/auth"
	"tinh-tien-api/internal/domain/inventory"
	"tinh-tien-api/internal/pkg/httputil"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page := httputil.ParsePageParams(r)
	q := ListQuery{
		Status:     Status(r.URL.Query().Get("status")),
		CustomerID: r.URL.Query().Get("customer_id"),
		Page:       page.Page,
		PageSize:   page.PageSize,
	}
	if from := r.URL.Query().Get("from"); from != "" {
		q.From = &from
	}
	if to := r.URL.Query().Get("to"); to != "" {
		q.To = &to
	}
	items, total, err := h.svc.List(q)
	if err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to list orders", err.Error())
		return
	}
	httputil.OKWithPagination(w, "orders retrieved", items, httputil.NewPagination(page.Page, page.PageSize, total))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	userID := auth.UserIDFromContext(r.Context())
	item, err := h.svc.Create(req, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmptyItems):
			httputil.Fail(w, http.StatusBadRequest, "invalid order", err.Error())
		case errors.Is(err, inventory.ErrInsufficient):
			httputil.Fail(w, http.StatusConflict, "insufficient stock", err.Error())
		default:
			httputil.Fail(w, http.StatusBadRequest, "failed to create order", err.Error())
		}
		return
	}
	httputil.Created(w, "order created", item)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	item, err := h.svc.Get(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.Fail(w, http.StatusNotFound, "order not found", err.Error())
			return
		}
		httputil.Fail(w, http.StatusInternalServerError, "failed to get order", err.Error())
		return
	}
	httputil.OK(w, "order retrieved", item)
}

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req UpdateStatusRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	item, err := h.svc.UpdateStatus(id, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			httputil.Fail(w, http.StatusNotFound, "order not found", err.Error())
		case errors.Is(err, ErrInvalidStatus):
			httputil.Fail(w, http.StatusBadRequest, "invalid status transition", err.Error())
		case errors.Is(err, inventory.ErrInsufficient):
			httputil.Fail(w, http.StatusConflict, "insufficient stock", err.Error())
		default:
			httputil.Fail(w, http.StatusInternalServerError, "failed to update order status", err.Error())
		}
		return
	}
	httputil.OK(w, "order status updated", item)
}

func (h *Handler) AddPayment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req CreatePaymentRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	userID := auth.UserIDFromContext(r.Context())
	item, err := h.svc.AddPayment(id, req, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			httputil.Fail(w, http.StatusNotFound, "order not found", err.Error())
		case errors.Is(err, ErrPaymentExceedsTotal), errors.Is(err, ErrInvalidStatus):
			httputil.Fail(w, http.StatusBadRequest, "invalid payment", err.Error())
		default:
			httputil.Fail(w, http.StatusInternalServerError, "failed to add payment", err.Error())
		}
		return
	}
	httputil.OK(w, "payment recorded", item)
}

func (h *Handler) ListReceivables(w http.ResponseWriter, r *http.Request) {
	page := httputil.ParsePageParams(r)
	items, total, err := h.svc.ListReceivables(page.Page, page.PageSize)
	if err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to list receivables", err.Error())
		return
	}
	httputil.OKWithPagination(w, "receivables retrieved", items, httputil.NewPagination(page.Page, page.PageSize, total))
}
