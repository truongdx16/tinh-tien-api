package product

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

func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	page := httputil.ParsePageParams(r)
	items, total, err := h.svc.ListCategories(page.Page, page.PageSize)
	if err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to list categories", err.Error())
		return
	}
	httputil.OKWithPagination(w, "categories retrieved", items, httputil.NewPagination(page.Page, page.PageSize, total))
}

func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	item, err := h.svc.CreateCategory(req)
	if err != nil {
		httputil.Fail(w, http.StatusBadRequest, "failed to create category", err.Error())
		return
	}
	httputil.Created(w, "category created", item)
}

func (h *Handler) GetCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	item, err := h.svc.GetCategory(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.Fail(w, http.StatusNotFound, "category not found", err.Error())
			return
		}
		httputil.Fail(w, http.StatusInternalServerError, "failed to get category", err.Error())
		return
	}
	httputil.OK(w, "category retrieved", item)
}

func (h *Handler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req UpdateCategoryRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	item, err := h.svc.UpdateCategory(id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.Fail(w, http.StatusNotFound, "category not found", err.Error())
			return
		}
		httputil.Fail(w, http.StatusInternalServerError, "failed to update category", err.Error())
		return
	}
	httputil.OK(w, "category updated", item)
}

func (h *Handler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteCategory(id); err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to delete category", err.Error())
		return
	}
	httputil.OK(w, "category deleted", nil)
}

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	page := httputil.ParsePageParams(r)
	q := ProductListQuery{
		CategoryID: r.URL.Query().Get("category_id"),
		Search:     r.URL.Query().Get("search"),
		Page:       page.Page,
		PageSize:   page.PageSize,
	}
	if activeStr := r.URL.Query().Get("active"); activeStr != "" {
		active, err := strconv.ParseBool(activeStr)
		if err == nil {
			q.Active = &active
		}
	}
	items, total, err := h.svc.ListProducts(q)
	if err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to list products", err.Error())
		return
	}
	httputil.OKWithPagination(w, "products retrieved", items, httputil.NewPagination(page.Page, page.PageSize, total))
}

func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req CreateProductRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	item, err := h.svc.CreateProduct(req)
	if err != nil {
		httputil.Fail(w, http.StatusBadRequest, "failed to create product", err.Error())
		return
	}
	httputil.Created(w, "product created", item)
}

func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	item, err := h.svc.GetProduct(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.Fail(w, http.StatusNotFound, "product not found", err.Error())
			return
		}
		httputil.Fail(w, http.StatusInternalServerError, "failed to get product", err.Error())
		return
	}
	httputil.OK(w, "product retrieved", item)
}

func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req UpdateProductRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	item, err := h.svc.UpdateProduct(id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.Fail(w, http.StatusNotFound, "product not found", err.Error())
			return
		}
		httputil.Fail(w, http.StatusInternalServerError, "failed to update product", err.Error())
		return
	}
	httputil.OK(w, "product updated", item)
}

func (h *Handler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteProduct(id); err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to delete product", err.Error())
		return
	}
	httputil.OK(w, "product deleted", nil)
}
