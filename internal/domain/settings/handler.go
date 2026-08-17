package settings

import (
	"net/http"

	"tinh-tien-api/internal/pkg/httputil"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	item, err := h.svc.GetShopSettingsDisplay()
	if err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to get settings", err.Error())
		return
	}
	httputil.OK(w, "settings retrieved", item)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdateSettingsRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	item, err := h.svc.UpdateShopSettings(req)
	if err != nil {
		httputil.Fail(w, http.StatusInternalServerError, "failed to update settings", err.Error())
		return
	}
	httputil.OK(w, "settings updated", item)
}
