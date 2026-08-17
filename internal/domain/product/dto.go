package product

type CreateCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateCategoryRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type CreateProductRequest struct {
	CategoryID  *string `json:"category_id"`
	Name        string  `json:"name"`
	Unit        string  `json:"unit"`
	SellPrice   int64   `json:"sell_price"`
	CostPrice   int64   `json:"cost_price"`
	Description string  `json:"description"`
	CropType    string  `json:"crop_type"`
	Seasonal    bool    `json:"seasonal"`
	Active      bool    `json:"active"`
}

type UpdateProductRequest struct {
	CategoryID  *string `json:"category_id"`
	Name        *string `json:"name"`
	Unit        *string `json:"unit"`
	SellPrice   *int64  `json:"sell_price"`
	CostPrice   *int64  `json:"cost_price"`
	Description *string `json:"description"`
	CropType    *string `json:"crop_type"`
	Seasonal    *bool   `json:"seasonal"`
	Active      *bool   `json:"active"`
}

type ProductListQuery struct {
	CategoryID string
	Active     *bool
	Search     string
	Page       int
	PageSize   int
}
