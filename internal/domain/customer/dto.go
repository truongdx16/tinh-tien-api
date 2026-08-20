package customer

type CreateCustomerRequest struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Address  string `json:"address"`
	Notes    string `json:"notes"`
	IsWalkIn bool   `json:"is_walk_in"`
}

type UpdateCustomerRequest struct {
	Code     *string `json:"code"`
	Name     *string `json:"name"`
	Phone    *string `json:"phone"`
	Address  *string `json:"address"`
	Notes    *string `json:"notes"`
	Active   *bool   `json:"active"`
	IsWalkIn *bool   `json:"is_walk_in"`
}

type ListQuery struct {
	Search   string
	Active   *bool
	Page     int
	PageSize int
}
