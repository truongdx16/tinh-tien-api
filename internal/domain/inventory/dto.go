package inventory

type AdjustmentRequest struct {
	ProductID string  `json:"product_id"`
	Quantity  float64 `json:"quantity"`
	Note      string  `json:"note"`
}

type ListQuery struct {
	LowStockThreshold float64
	Page              int
	PageSize          int
}

type BalanceResponse struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Unit        string  `json:"unit"`
	Quantity    float64 `json:"quantity"`
	LowStock    bool    `json:"low_stock"`
}
