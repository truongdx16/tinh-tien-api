package expense

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"tinh-tien-api/internal/domain/auth"
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
		From: r.URL.Query().Get("from"), To: r.URL.Query().Get("to"),
		Category: Category(r.URL.Query().Get("category")),
		Page: page.Page, PageSize: page.PageSize,
	}
	items, total, err := h.svc.List(q)
	if err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to list expenses", err.Error())
		return
	}
	httputil.OKWithPagination(w, "expenses retrieved", items, httputil.NewPagination(page.Page, page.PageSize, total))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateExpenseRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	userID := auth.UserIDFromContext(r.Context())
	item, err := h.svc.Create(req, userID)
	if err != nil {
		httputil.Fail(w, http.StatusBadRequest, "failed to create expense", err.Error())
		return
	}
	httputil.Created(w, "expense created", item)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	item, err := h.svc.Get(chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.Fail(w, http.StatusNotFound, "expense not found", err.Error())
			return
		}
		httputil.Fail(w, http.StatusInternalServerError, "failed to get expense", err.Error())
		return
	}
	httputil.OK(w, "expense retrieved", item)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdateExpenseRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	item, err := h.svc.Update(chi.URLParam(r, "id"), req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.Fail(w, http.StatusNotFound, "expense not found", err.Error())
			return
		}
		httputil.Fail(w, http.StatusInternalServerError, "failed to update expense", err.Error())
		return
	}
	httputil.OK(w, "expense updated", item)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(chi.URLParam(r, "id")); err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to delete expense", err.Error())
		return
	}
	httputil.OK(w, "expense deleted", nil)
}
