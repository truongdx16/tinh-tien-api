package settings

type ShopSettings struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Currency string `json:"currency"`
}

type UpdateSettingsRequest struct {
	Name     *string `json:"name"`
	Phone    *string `json:"phone"`
	Currency *string `json:"currency"`
}
