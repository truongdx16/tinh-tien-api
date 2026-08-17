package settings

import (
	"tinh-tien-api/internal/pkg/model"
)

type Setting struct {
	model.Base
	Key   string `gorm:"column:setting_key;size:64;uniqueIndex:idx_settings_setting_key;not null" json:"key"`
	Value string `gorm:"size:1024;not null" json:"value"`
}

const (
	KeyShopName     = "shop_name"
	KeyShopPhone    = "shop_phone"
	KeyCurrency     = "currency"
)
