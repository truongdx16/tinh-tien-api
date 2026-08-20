package order

import (
	"time"

	"tinh-tien-api/internal/domain/customer"
	"tinh-tien-api/internal/pkg/model"
)

type Status string

const (
	StatusDraft     Status = "draft"
	StatusConfirmed Status = "confirmed"
	StatusPacked    Status = "packed"
	StatusDelivered Status = "delivered"
	StatusCancelled Status = "cancelled"
)

type FulfillmentType string

const (
	FulfillmentPickup   FulfillmentType = "pickup"
	FulfillmentDelivery FulfillmentType = "delivery"
)

type PaymentMethod string

const (
	PaymentCash   PaymentMethod = "cash"
	PaymentBank   PaymentMethod = "bank_transfer"
	PaymentCOD    PaymentMethod = "cod"
)

type Order struct {
	model.Base
	CustomerID      *string         `gorm:"type:uuid;index" json:"customer_id"`
	Customer        *customer.Customer `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Status          Status          `gorm:"size:32;not null;default:draft" json:"status"`
	FulfillmentType FulfillmentType `gorm:"size:32;not null;default:pickup" json:"fulfillment_type"`
	DeliveryAddress string          `gorm:"size:512" json:"delivery_address"`
	Note            string          `gorm:"size:1024" json:"note"`
	AllowBackorder  bool            `gorm:"not null;default:false" json:"allow_backorder"`
	Discount        int64           `gorm:"not null;default:0" json:"discount"`
	Subtotal        int64           `gorm:"not null;default:0" json:"subtotal"`
	Total           int64           `gorm:"not null;default:0" json:"total"`
	PaidAmount      int64           `gorm:"not null;default:0" json:"paid_amount"`
	BalanceDue      int64           `gorm:"not null;default:0" json:"balance_due"`
	CreatedBy       string          `gorm:"type:uuid" json:"created_by"`
	Items           []OrderItem     `gorm:"foreignKey:OrderID" json:"items,omitempty"`
	Payments        []Payment       `gorm:"foreignKey:OrderID" json:"payments,omitempty"`
}

type OrderItem struct {
	model.Base
	OrderID   string  `gorm:"type:uuid;index;not null" json:"order_id"`
	ProductID string  `gorm:"type:uuid;index;not null" json:"product_id"`
	Name      string  `gorm:"size:256;not null" json:"name"`
	Unit      string  `gorm:"size:32;not null" json:"unit"`
	Quantity  float64 `gorm:"not null" json:"quantity"`
	UnitPrice int64   `gorm:"not null" json:"unit_price"`
	LineTotal int64   `gorm:"not null" json:"line_total"`
}

type Payment struct {
	model.Base
	OrderID   string        `gorm:"type:uuid;index;not null" json:"order_id"`
	Amount    int64         `gorm:"not null" json:"amount"`
	Method    PaymentMethod `gorm:"size:32;not null" json:"method"`
	Note      string        `gorm:"size:512" json:"note"`
	PaidAt    time.Time     `gorm:"not null" json:"paid_at"`
	CreatedBy string        `gorm:"type:uuid" json:"created_by"`
}
