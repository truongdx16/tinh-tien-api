package crop

import (
	"errors"
	"net/http"
	"strconv"

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

func (h *Handler) ListPlots(w http.ResponseWriter, r *http.Request) {
	page := httputil.ParsePageParams(r)
	items, total, err := h.svc.ListPlots(page.Page, page.PageSize)
	if err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to list plots", err.Error())
		return
	}
	httputil.OKWithPagination(w, "plots retrieved", items, httputil.NewPagination(page.Page, page.PageSize, total))
}

func (h *Handler) CreatePlot(w http.ResponseWriter, r *http.Request) {
	var req CreatePlotRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	item, err := h.svc.CreatePlot(req)
	if err != nil {
		httputil.Fail(w, http.StatusBadRequest, "failed to create plot", err.Error())
		return
	}
	httputil.Created(w, "plot created", item)
}

func (h *Handler) GetPlot(w http.ResponseWriter, r *http.Request) {
	item, err := h.svc.GetPlot(chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.Fail(w, http.StatusNotFound, "plot not found", err.Error())
			return
		}
		httputil.Fail(w, http.StatusInternalServerError, "failed to get plot", err.Error())
		return
	}
	httputil.OK(w, "plot retrieved", item)
}

func (h *Handler) UpdatePlot(w http.ResponseWriter, r *http.Request) {
	var req UpdatePlotRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	item, err := h.svc.UpdatePlot(chi.URLParam(r, "id"), req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.Fail(w, http.StatusNotFound, "plot not found", err.Error())
			return
		}
		httputil.Fail(w, http.StatusInternalServerError, "failed to update plot", err.Error())
		return
	}
	httputil.OK(w, "plot updated", item)
}

func (h *Handler) DeletePlot(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeletePlot(chi.URLParam(r, "id")); err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to delete plot", err.Error())
		return
	}
	httputil.OK(w, "plot deleted", nil)
}

func (h *Handler) ListBatches(w http.ResponseWriter, r *http.Request) {
	page := httputil.ParsePageParams(r)
	q := BatchListQuery{
		PlotID: r.URL.Query().Get("plot_id"),
		Status: BatchStatus(r.URL.Query().Get("status")),
		Page:   page.Page, PageSize: page.PageSize,
	}
	items, total, err := h.svc.ListBatches(q)
	if err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to list batches", err.Error())
		return
	}
	httputil.OKWithPagination(w, "crop batches retrieved", items, httputil.NewPagination(page.Page, page.PageSize, total))
}

func (h *Handler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	var req CreateBatchRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	item, err := h.svc.CreateBatch(req)
	if err != nil {
		httputil.Fail(w, http.StatusBadRequest, "failed to create batch", err.Error())
		return
	}
	httputil.Created(w, "crop batch created", item)
}

func (h *Handler) GetBatch(w http.ResponseWriter, r *http.Request) {
	item, err := h.svc.GetBatch(chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.Fail(w, http.StatusNotFound, "batch not found", err.Error())
			return
		}
		httputil.Fail(w, http.StatusInternalServerError, "failed to get batch", err.Error())
		return
	}
	httputil.OK(w, "crop batch retrieved", item)
}

func (h *Handler) UpdateBatch(w http.ResponseWriter, r *http.Request) {
	var req UpdateBatchRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	item, err := h.svc.UpdateBatch(chi.URLParam(r, "id"), req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.Fail(w, http.StatusNotFound, "batch not found", err.Error())
			return
		}
		httputil.Fail(w, http.StatusInternalServerError, "failed to update batch", err.Error())
		return
	}
	httputil.OK(w, "crop batch updated", item)
}

func (h *Handler) AddActivity(w http.ResponseWriter, r *http.Request) {
	var req CreateActivityRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	userID := auth.UserIDFromContext(r.Context())
	item, err := h.svc.AddActivity(chi.URLParam(r, "id"), req, userID)
	if err != nil {
		httputil.Fail(w, http.StatusBadRequest, "failed to add activity", err.Error())
		return
	}
	httputil.Created(w, "activity recorded", item)
}

func (h *Handler) RecordHarvest(w http.ResponseWriter, r *http.Request) {
	var req CreateHarvestRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	userID := auth.UserIDFromContext(r.Context())
	item, err := h.svc.RecordHarvest(chi.URLParam(r, "id"), req, userID)
	if err != nil {
		httputil.Fail(w, http.StatusBadRequest, "failed to record harvest", err.Error())
		return
	}
	httputil.Created(w, "harvest recorded", item)
}

func (h *Handler) DueHarvests(w http.ResponseWriter, r *http.Request) {
	page := httputil.ParsePageParams(r)
	days := 7
	if v := r.URL.Query().Get("days"); v != "" {
		if d, err := strconv.Atoi(v); err == nil {
			days = d
		}
	}
	items, total, err := h.svc.ListDueHarvests(days, page.Page, page.PageSize)
	if err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to list due harvests", err.Error())
		return
	}
	httputil.OKWithPagination(w, "due harvests retrieved", items, httputil.NewPagination(page.Page, page.PageSize, total))
}
