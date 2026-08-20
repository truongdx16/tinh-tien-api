package mobile

import (
	"errors"
	"net/http"
	"strconv"

	"tinh-tien-api/internal/domain/auth"
	"tinh-tien-api/internal/domain/order"
	"tinh-tien-api/internal/mobile/adapter"
	"tinh-tien-api/internal/pkg/httputil"

	"github.com/go-chi/chi/v5"
)

// Flutter order status int mapping:
// 0 = pending/draft, 1 = processing/confirmed, 2 = completed/delivered, 3 = cancelled

var flutterToGoStatus = map[int]order.Status{
	0: order.StatusDraft,
	1: order.StatusConfirmed,
	2: order.StatusDelivered,
	3: order.StatusCancelled,
}

var goToFlutterStatus = map[order.Status]int{
	order.StatusDraft:     0,
	order.StatusConfirmed: 1,
	order.StatusPacked:    1,
	order.StatusDelivered: 2,
	order.StatusCancelled: 3,
}

type MobileOrderItemDto struct {
	ProductID   *string `json:"product_id"`
	ProductName string  `json:"product_name"`
	UnitName    *string `json:"unit_name"`
	UnitPrice   string  `json:"unit_price"`
	Quantity    string  `json:"quantity"`
	LineTotal   string  `json:"line_total"`
}

type MobileOrderDto struct {
	ID         string               `json:"id"`
	CustomerID *string              `json:"customer_id"`
	Discount   string               `json:"discount"`
	Revenue    string               `json:"revenue"`
	PaidAmount string               `json:"paid_amount"`
	Status     int                  `json:"status"`
	CreatedAt  *string              `json:"created_at"`
	UpdatedAt  *string              `json:"updated_at"`
	Customer   *MobileCustomerDto   `json:"customer"`
	Items      []MobileOrderItemDto `json:"items"`
}

type MobileOrderWriteBody struct {
	ID         *string `json:"id"`
	CustomerID *string `json:"customer_id"`
	Discount   float64 `json:"discount"`
	PaidAmount float64 `json:"paid_amount"`
	Items      []struct {
		ProductID string  `json:"product_id"`
		Quantity  float64 `json:"quantity"`
	} `json:"items"`
}

type OrderMobileHandler struct {
	svc *order.Service
}

func NewOrderMobileHandler(svc *order.Service) *OrderMobileHandler {
	return &OrderMobileHandler{svc: svc}
}

func (h *OrderMobileHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from := q.Get("from")
	to := q.Get("to")

	listQ := order.ListQuery{
		From:     &from,
		To:       &to,
		Page:     1,
		PageSize: 200,
	}
	if s := q.Get("status"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			if goStat, ok := flutterToGoStatus[v]; ok {
				listQ.Status = goStat
			}
		}
	}
	items, _, err := h.svc.List(listQ)
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "failed to list orders")
		return
	}
	dtos := make([]MobileOrderDto, 0, len(items))
	for _, o := range items {
		dtos = append(dtos, toOrderDto(o))
	}
	adapter.OK(w, "orders retrieved", dtos)
}

func (h *OrderMobileHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	o, err := h.svc.Get(id)
	if err != nil {
		if errors.Is(err, order.ErrNotFound) {
			adapter.Fail(w, http.StatusNotFound, "order not found")
			return
		}
		adapter.Fail(w, http.StatusInternalServerError, "failed to get order")
		return
	}
	adapter.OK(w, "order retrieved", toOrderDto(*o))
}

func (h *OrderMobileHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body MobileOrderWriteBody
	if err := httputil.Decode(r, &body); err != nil {
		adapter.Fail(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userID := auth.UserIDFromContext(r.Context())

	items := make([]order.OrderItemRequest, 0, len(body.Items))
	for _, it := range body.Items {
		items = append(items, order.OrderItemRequest{
			ProductID: it.ProductID,
			Quantity:  it.Quantity,
		})
	}

	o, err := h.svc.Create(order.CreateOrderRequest{
		CustomerID: body.CustomerID,
		Items:      items,
		Status:     order.StatusConfirmed, // Flutter creates orders as confirmed
	}, userID)
	if err != nil {
		adapter.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	// Apply discount if present
	if body.Discount > 0 {
		// Update discount field on order
		disc := int64(body.Discount)
		o.Discount = disc
		o.Total = o.Subtotal - disc
		if o.Total < 0 {
			o.Total = 0
		}
		o.BalanceDue = o.Total - o.PaidAmount
	}
	adapter.Created(w, "order created", toOrderDto(*o))
}

func (h *OrderMobileHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	o, err := h.svc.UpdateStatus(id, order.UpdateStatusRequest{Status: order.StatusCancelled})
	if err != nil {
		if errors.Is(err, order.ErrNotFound) {
			adapter.Fail(w, http.StatusNotFound, "order not found")
			return
		}
		adapter.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	adapter.OK(w, "order cancelled", toOrderDto(*o))
}

func toOrderDto(o order.Order) MobileOrderDto {
	items := make([]MobileOrderItemDto, 0, len(o.Items))
	for _, it := range o.Items {
		pid := it.ProductID
		unitName := it.Unit
		items = append(items, MobileOrderItemDto{
			ProductID:   &pid,
			ProductName: it.Name,
			UnitName:    &unitName,
			UnitPrice:   adapter.MoneyStr(it.UnitPrice),
			Quantity:    adapter.FloatStr(it.Quantity),
			LineTotal:   adapter.MoneyStr(it.LineTotal),
		})
	}

	createdAt := o.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	updatedAt := o.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")

	var custDto *MobileCustomerDto
	if o.Customer != nil {
		d := toCustomerDto(*o.Customer)
		custDto = &d
	}

	statusInt := goToFlutterStatus[o.Status]

	return MobileOrderDto{
		ID:         o.ID.String(),
		CustomerID: o.CustomerID,
		Discount:   adapter.MoneyStr(o.Discount),
		Revenue:    adapter.MoneyStr(o.Total),
		PaidAmount: adapter.MoneyStr(o.PaidAmount),
		Status:     statusInt,
		CreatedAt:  &createdAt,
		UpdatedAt:  &updatedAt,
		Customer:   custDto,
		Items:      items,
	}
}
