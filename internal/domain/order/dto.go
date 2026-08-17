package order

type OrderItemRequest struct {
	ProductID string  `json:"product_id"`
	Quantity  float64 `json:"quantity"`
	UnitPrice *int64  `json:"unit_price"`
}

type CreateOrderRequest struct {
	CustomerID      *string             `json:"customer_id"`
	FulfillmentType FulfillmentType     `json:"fulfillment_type"`
	DeliveryAddress string              `json:"delivery_address"`
	Note            string              `json:"note"`
	AllowBackorder  bool                `json:"allow_backorder"`
	Items           []OrderItemRequest  `json:"items"`
	Status          Status              `json:"status"`
}

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
	Page       int
	PageSize   int
}

type ReceivableItem struct {
	CustomerID   string `json:"customer_id"`
	CustomerName string `json:"customer_name"`
	OrderID      string `json:"order_id"`
	BalanceDue   int64  `json:"balance_due"`
	Status       Status `json:"status"`
}
