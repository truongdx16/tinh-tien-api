package mobile

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tinh-tien-api/internal/domain/auth"
	"tinh-tien-api/internal/domain/customer"
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
	ID         *string  `json:"id"`
	CustomerID *string  `json:"customer_id"`
	Discount   float64  `json:"discount"`
	PaidAmount *float64 `json:"paid_amount"` // omitted by Flutter when paid in full
	Items      []struct {
		ProductID string  `json:"product_id"`
		Quantity  float64 `json:"quantity"`
	} `json:"items"`
}

func (b MobileOrderWriteBody) orderItems() []order.OrderItemRequest {
	items := make([]order.OrderItemRequest, 0, len(b.Items))
	for _, it := range b.Items {
		items = append(items, order.OrderItemRequest{
			ProductID: it.ProductID,
			Quantity:  it.Quantity,
		})
	}
	return items
}

func (b MobileOrderWriteBody) paidAmount() *int64 {
	if b.PaidAmount == nil {
		return nil
	}
	v := int64(math.Round(*b.PaidAmount))
	return &v
}

type OrderMobileHandler struct {
	svc         *order.Service
	customerSvc *customer.Service
	loc         *time.Location
}

func NewOrderMobileHandler(svc *order.Service, customerSvc *customer.Service, loc *time.Location) *OrderMobileHandler {
	return &OrderMobileHandler{svc: svc, customerSvc: customerSvc, loc: loc}
}

// maxOrdersPerList bounds GET /orders (Flutter has no paging for orders; ~30 orders/day).
const maxOrdersPerList = 20000

func (h *OrderMobileHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	fromT, toT, err := adapter.DayRange(q.Get("from"), q.Get("to"), h.loc)
	if err != nil {
		adapter.Fail(w, http.StatusBadRequest, userMessage(err, "Ngày không hợp lệ"))
		return
	}

	listQ := order.ListQuery{
		FromTime: fromT,
		ToTime:   toT,
		Page:     1,
		PageSize: maxOrdersPerList,
	}
	if s := q.Get("status"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			if goStat, ok := flutterToGoStatus[v]; ok {
				listQ.Status = goStat
			}
		}
	}
	// customer_id: orders of one customer (uuid or the walk-in code).
	if c := strings.TrimSpace(q.Get("customer_id")); c != "" {
		id, err := h.customerSvc.ResolveID(c)
		if err != nil {
			adapter.Fail(w, http.StatusNotFound, "Không tìm thấy khách hàng")
			return
		}
		listQ.CustomerID = id
	}
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v < listQ.PageSize {
			listQ.PageSize = v
		}
	}
	items, _, err := h.svc.List(listQ)
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "Không tải được danh sách đơn hàng")
		return
	}
	dtos := make([]MobileOrderDto, 0, len(items))
	for _, o := range items {
		dtos = append(dtos, h.toOrderDto(o))
	}
	adapter.OK(w, "orders retrieved", dtos)
}

// Delete: DELETE /orders/{id}, owner (admin) only. The order leaves history
// and statistics; stock it took out goes back.
func (h *OrderMobileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := auth.FromContext(r.Context())
	if claims == nil || claims.Role != auth.RoleOwner {
		adapter.Fail(w, http.StatusForbidden, "Chỉ quản trị viên mới được xoá đơn hàng")
		return
	}
	if err := h.svc.DeleteOrder(chi.URLParam(r, "id")); err != nil {
		if errors.Is(err, order.ErrNotFound) {
			adapter.Fail(w, http.StatusNotFound, "Không tìm thấy đơn hàng")
			return
		}
		adapter.Fail(w, http.StatusInternalServerError, "Không xoá được đơn hàng")
		return
	}
	adapter.OK(w, "Đã xoá đơn hàng", nil)
}

func (h *OrderMobileHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	o, err := h.svc.Get(id)
	if err != nil {
		if errors.Is(err, order.ErrNotFound) {
			adapter.Fail(w, http.StatusNotFound, "Không tìm thấy đơn hàng")
			return
		}
		adapter.Fail(w, http.StatusInternalServerError, "Không tải được đơn hàng")
		return
	}
	adapter.OK(w, "order retrieved", h.toOrderDto(*o))
}

func (h *OrderMobileHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body MobileOrderWriteBody
	if err := httputil.Decode(r, &body); err != nil {
		adapter.Fail(w, http.StatusBadRequest, "Dữ liệu gửi lên không hợp lệ")
		return
	}
	userID := auth.UserIDFromContext(r.Context())

	// Flutter sends "CUS-0001" for walk-in sales → resolve to the customer UUID.
	var customerID *string
	if body.CustomerID != nil && *body.CustomerID != "" {
		id, err := h.customerSvc.ResolveID(*body.CustomerID)
		if err != nil {
			adapter.Fail(w, http.StatusBadRequest, "Không tìm thấy khách hàng")
			return
		}
		customerID = &id
	}

	o, err := h.svc.Create(order.CreateOrderRequest{
		CustomerID: customerID,
		Items:      body.orderItems(),
		// POS sale: completed immediately (matches imported history, Flutter status 2).
		// Stock is not tracked by the mobile app, so no stock check / deduction.
		Status:         order.StatusDelivered,
		AllowBackorder: true,
		Discount:       int64(math.Round(body.Discount)),
		PaidAmount:     body.paidAmount(),
	}, userID)
	if err != nil {
		adapter.Fail(w, http.StatusBadRequest, userMessage(err, "Không tạo được đơn hàng"))
		return
	}
	adapter.Created(w, "order created", h.toOrderDto(*o))
}

// Update handles PUT /orders/{id}: edits the items, discount and payment of an
// order that is not completed or cancelled.
func (h *OrderMobileHandler) Update(w http.ResponseWriter, r *http.Request) {
	var body MobileOrderWriteBody
	if err := httputil.Decode(r, &body); err != nil {
		adapter.Fail(w, http.StatusBadRequest, "Dữ liệu gửi lên không hợp lệ")
		return
	}
	o, err := h.svc.Update(chi.URLParam(r, "id"), order.UpdateOrderRequest{
		Items:      body.orderItems(),
		Discount:   int64(math.Round(body.Discount)),
		PaidAmount: body.paidAmount(),
	})
	if err != nil {
		h.failWrite(w, err, "Không cập nhật được đơn hàng")
		return
	}
	adapter.OK(w, "order updated", h.toOrderDto(*o))
}

// UpdateStatus handles PATCH /orders/{id}/status with {"status": 0-3}.
func (h *OrderMobileHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status *int `json:"status"`
	}
	if err := httputil.Decode(r, &body); err != nil || body.Status == nil {
		adapter.Fail(w, http.StatusBadRequest, "Dữ liệu gửi lên không hợp lệ")
		return
	}
	status, ok := flutterToGoStatus[*body.Status]
	if !ok {
		adapter.Fail(w, http.StatusBadRequest, "Trạng thái đơn hàng không hợp lệ")
		return
	}
	o, err := h.svc.UpdateStatus(chi.URLParam(r, "id"), order.UpdateStatusRequest{Status: status})
	if err != nil {
		h.failWrite(w, err, "Không cập nhật được trạng thái đơn hàng")
		return
	}
	adapter.OK(w, "order status updated", h.toOrderDto(*o))
}

func (h *OrderMobileHandler) failWrite(w http.ResponseWriter, err error, fallback string) {
	status := http.StatusBadRequest
	if errors.Is(err, order.ErrNotFound) {
		status = http.StatusNotFound
	} else if errors.Is(err, order.ErrInvalidStatus) || errors.Is(err, order.ErrOrderClosed) {
		status = http.StatusConflict
	}
	adapter.Fail(w, status, userMessage(err, fallback))
}

func (h *OrderMobileHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	o, err := h.svc.UpdateStatus(id, order.UpdateStatusRequest{Status: order.StatusCancelled})
	if err != nil {
		if errors.Is(err, order.ErrNotFound) {
			adapter.Fail(w, http.StatusNotFound, "Không tìm thấy đơn hàng")
			return
		}
		adapter.Fail(w, http.StatusBadRequest, userMessage(err, "Không huỷ được đơn hàng"))
		return
	}
	adapter.OK(w, "order cancelled", h.toOrderDto(*o))
}

func (h *OrderMobileHandler) toOrderDto(o order.Order) MobileOrderDto {
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

	createdAt := adapter.ISOTime(o.CreatedAt, h.loc)
	updatedAt := adapter.ISOTime(o.UpdatedAt, h.loc)

	customerID := o.CustomerID
	var custDto *MobileCustomerDto
	if o.Customer != nil {
		d := toCustomerDto(*o.Customer)
		custDto = &d
		customerID = &d.ID
	}

	statusInt := goToFlutterStatus[o.Status]

	return MobileOrderDto{
		ID:         o.ID.String(),
		CustomerID: customerID,
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
