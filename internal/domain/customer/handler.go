package customer

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
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
	q := ListQuery{Search: r.URL.Query().Get("search"), Page: page.Page, PageSize: page.PageSize}
	if activeStr := r.URL.Query().Get("active"); activeStr != "" {
		active, err := strconv.ParseBool(activeStr)
		if err == nil {
			q.Active = &active
		}
	}
	items, total, err := h.svc.List(q)
	if err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to list customers", err.Error())
		return
	}
	httputil.OKWithPagination(w, "customers retrieved", items, httputil.NewPagination(page.Page, page.PageSize, total))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateCustomerRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	item, err := h.svc.Create(req)
	if err != nil {
		httputil.Fail(w, http.StatusBadRequest, "failed to create customer", err.Error())
		return
	}
	httputil.Created(w, "customer created", item)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	item, err := h.svc.Get(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.Fail(w, http.StatusNotFound, "customer not found", err.Error())
			return
		}
		httputil.Fail(w, http.StatusInternalServerError, "failed to get customer", err.Error())
		return
	}
	httputil.OK(w, "customer retrieved", item)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req UpdateCustomerRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	item, err := h.svc.Update(id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.Fail(w, http.StatusNotFound, "customer not found", err.Error())
			return
		}
		httputil.Fail(w, http.StatusInternalServerError, "failed to update customer", err.Error())
		return
	}
	httputil.OK(w, "customer updated", item)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(id); err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to delete customer", err.Error())
		return
	}
	httputil.OK(w, "customer deleted", nil)
}
