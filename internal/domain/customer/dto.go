package customer

type CreateCustomerRequest struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
	Notes   string `json:"notes"`
}

type UpdateCustomerRequest struct {
	Name    *string `json:"name"`
	Phone   *string `json:"phone"`
	Address *string `json:"address"`
	Notes   *string `json:"notes"`
	Active  *bool   `json:"active"`
}

type ListQuery struct {
	Search   string
	Active   *bool
	Page     int
	PageSize int
}
