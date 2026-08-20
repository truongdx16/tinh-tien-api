package mobile

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"tinh-tien-api/internal/domain/customer"
	"tinh-tien-api/internal/mobile/adapter"
	"tinh-tien-api/internal/pkg/httputil"
)

type MobileCustomerDto struct {
	ID       string  `json:"id"`
	Code     *string `json:"code"`
	Name     string  `json:"name"`
	Phone    *string `json:"phone"`
	Address  *string `json:"address"`
	Status   int     `json:"status"`
	IsWalkIn bool    `json:"is_walk_in"`
}

type CustomerMobileHandler struct {
	svc *customer.Service
}

func NewCustomerMobileHandler(svc *customer.Service) *CustomerMobileHandler {
	return &CustomerMobileHandler{svc: svc}
}

func (h *CustomerMobileHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page := 1
	perPage := 20
	if p := q.Get("per_page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			perPage = v
		}
	}
	listQ := customer.ListQuery{Search: q.Get("search"), Page: page, PageSize: perPage}
	items, total, err := h.svc.List(listQ)
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "failed to list customers")
		return
	}
	dtos := make([]MobileCustomerDto, 0, len(items))
	for _, c := range items {
		dtos = append(dtos, toCustomerDto(c))
	}
	adapter.OKPaginated(w, "customers retrieved", dtos, page, total, perPage)
}

func (h *CustomerMobileHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code     *string `json:"code"`
		Name     string  `json:"name"`
		Phone    *string `json:"phone"`
		Address  *string `json:"address"`
		Status   int     `json:"status"`
		IsWalkIn bool    `json:"is_walk_in"`
	}
	if err := httputil.Decode(r, &body); err != nil {
		adapter.Fail(w, http.StatusBadRequest, "invalid request body")
		return
	}
	code := ""
	if body.Code != nil {
		code = *body.Code
	}
	phone := ""
	if body.Phone != nil {
		phone = *body.Phone
	}
	address := ""
	if body.Address != nil {
		address = *body.Address
	}
	c, err := h.svc.Create(customer.CreateCustomerRequest{
		Code:     code,
		Name:     body.Name,
		Phone:    phone,
		Address:  address,
		IsWalkIn: body.IsWalkIn,
	})
	if err != nil {
		adapter.Fail(w, http.StatusBadRequest, "failed to create customer")
		return
	}
	adapter.Created(w, "customer created", toCustomerDto(*c))
}

func (h *CustomerMobileHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Code     *string `json:"code"`
		Name     *string `json:"name"`
		Phone    *string `json:"phone"`
		Address  *string `json:"address"`
		Status   *int    `json:"status"`
		IsWalkIn *bool   `json:"is_walk_in"`
	}
	if err := httputil.Decode(r, &body); err != nil {
		adapter.Fail(w, http.StatusBadRequest, "invalid request body")
		return
	}
	var active *bool
	if body.Status != nil {
		a := *body.Status == 1
		active = &a
	}
	c, err := h.svc.Update(id, customer.UpdateCustomerRequest{
		Code:     body.Code,
		Name:     body.Name,
		Phone:    body.Phone,
		Address:  body.Address,
		Active:   active,
		IsWalkIn: body.IsWalkIn,
	})
	if err != nil {
		if errors.Is(err, customer.ErrNotFound) {
			adapter.Fail(w, http.StatusNotFound, "customer not found")
			return
		}
		adapter.Fail(w, http.StatusInternalServerError, "failed to update customer")
		return
	}
	adapter.OK(w, "customer updated", toCustomerDto(*c))
}

func toCustomerDto(c customer.Customer) MobileCustomerDto {
	var code, phone, address *string
	if c.Code != "" {
		code = &c.Code
	}
	if c.Phone != "" {
		phone = &c.Phone
	}
	if c.Address != "" {
		address = &c.Address
	}
	return MobileCustomerDto{
		ID:       c.ID.String(),
		Code:     code,
		Name:     c.Name,
		Phone:    phone,
		Address:  address,
		Status:   adapter.BoolToStatus(c.Active),
		IsWalkIn: c.IsWalkIn,
	}
}
