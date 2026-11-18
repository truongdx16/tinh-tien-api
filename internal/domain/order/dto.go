package order

import "time"

type OrderItemRequest struct {
	ProductID string  `json:"product_id"`
	Quantity  float64 `json:"quantity"`
	UnitPrice *int64  `json:"unit_price"`
}

// UpdateOrderRequest replaces the lines, discount and payment of an open order.
type UpdateOrderRequest struct {
	Items    []OrderItemRequest `json:"items"`
	Discount int64              `json:"discount"`
	// PaidAmount nil = paid in full.
	PaidAmount *int64 `json:"paid_amount"`
}

type CreateOrderRequest struct {
	CustomerID      *string            `json:"customer_id"`
	FulfillmentType FulfillmentType    `json:"fulfillment_type"`
	DeliveryAddress string             `json:"delivery_address"`
	Note            string             `json:"note"`
	AllowBackorder  bool               `json:"allow_backorder"`
	Items           []OrderItemRequest `json:"items"`
	Status          Status             `json:"status"`
	Discount        int64              `json:"discount"`
	// PaidAmount nil = paid in full. Set when the customer pays partially (debt = total - paid).
	PaidAmount *int64 `json:"paid_amount"`
	// ID chosen by the client (uuid) so a retried request finds the order it
	// already created instead of making a second one. Optional.
	ID *string `json:"id"`
	// CreatedAt of a sale made offline and sent later; must be within
	// MaxBackdate. Optional (default: now).
	CreatedAt *time.Time `json:"created_at"`
}

// MaxBackdate is how old an offline sale may be when it is finally sent.
const MaxBackdate = 30 * 24 * time.Hour

type UpdateStatusRequest struct {
	Status Status `json:"status"`
}

type CreatePaymentRequest struct {
	Amount int64         `json:"amount"`
	Method PaymentMethod `json:"method"`
	Note   string        `json:"note"`
}

type ListQuery struct {
	Status     Status
	CustomerID string
	From       *string
	To         *string
	// FromTime/ToTime (UTC, half-open [FromTime, ToTime)) take precedence over From/To.
	FromTime *time.Time
	ToTime   *time.Time
	Page     int
	PageSize int
}

type ReceivableItem struct {
	CustomerID   string `json:"customer_id"`
	CustomerName string `json:"customer_name"`
	OrderID      string `json:"order_id"`
	BalanceDue   int64  `json:"balance_due"`
	Status       Status `json:"status"`
}
