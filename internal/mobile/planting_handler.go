package mobile

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"tinh-tien-api/internal/domain/planting"
	"tinh-tien-api/internal/mobile/adapter"
	"tinh-tien-api/internal/pkg/httputil"
)

type PlantingMobileHandler struct {
	svc *planting.Service
}

func NewPlantingMobileHandler(svc *planting.Service) *PlantingMobileHandler {
	return &PlantingMobileHandler{svc: svc}
}

func (h *PlantingMobileHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List()
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "failed to list planting schedules")
		return
	}
	adapter.OK(w, "planting schedules retrieved", items)
}

func (h *PlantingMobileHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	item, err := h.svc.Get(id)
	if err != nil {
		if errors.Is(err, planting.ErrNotFound) {
			adapter.Fail(w, http.StatusNotFound, "planting schedule not found")
			return
		}
		adapter.Fail(w, http.StatusInternalServerError, "failed to get planting schedule")
		return
	}
	adapter.OK(w, "planting schedule retrieved", item)
}

func (h *PlantingMobileHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req planting.WriteRequest
	if err := httputil.Decode(r, &req); err != nil {
		adapter.Fail(w, http.StatusBadRequest, "invalid request body")
		return
	}
	item, err := h.svc.Create(req)
	if err != nil {
		adapter.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	adapter.Created(w, "planting schedule created", item)
}

func (h *PlantingMobileHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req planting.WriteRequest
	if err := httputil.Decode(r, &req); err != nil {
		adapter.Fail(w, http.StatusBadRequest, "invalid request body")
		return
	}
	item, err := h.svc.Update(id, req)
	if err != nil {
		if errors.Is(err, planting.ErrNotFound) {
			adapter.Fail(w, http.StatusNotFound, "planting schedule not found")
			return
		}
		adapter.Fail(w, http.StatusInternalServerError, "failed to update planting schedule")
		return
	}
	adapter.OK(w, "planting schedule updated", item)
}

func (h *PlantingMobileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(id); err != nil {
		if errors.Is(err, planting.ErrNotFound) {
			adapter.Fail(w, http.StatusNotFound, "planting schedule not found")
			return
		}
		adapter.Fail(w, http.StatusInternalServerError, "failed to delete planting schedule")
		return
	}
	adapter.OK(w, "planting schedule deleted", nil)
}
