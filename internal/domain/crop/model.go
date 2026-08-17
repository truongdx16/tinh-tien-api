package crop

import (
	"time"

	"tinh-tien-api/internal/pkg/model"
)

type PlotStatus string

const (
	PlotActive   PlotStatus = "active"
	PlotInactive PlotStatus = "inactive"
)

type BatchStatus string

const (
	BatchPlanned   BatchStatus = "planned"
	BatchGrowing   BatchStatus = "growing"
	BatchHarvested BatchStatus = "harvested"
	BatchFailed    BatchStatus = "failed"
)

type ActivityType string

const (
	ActivitySow       ActivityType = "sow"
	ActivityWater   ActivityType = "water"
	ActivityFertilize ActivityType = "fertilize"
	ActivityPest    ActivityType = "pest"
	ActivityHarvest ActivityType = "harvest"
	ActivityOther   ActivityType = "other"
)

type Plot struct {
	model.Base
	Name        string     `gorm:"size:128;not null" json:"name"`
	Location    string     `gorm:"size:256" json:"location"`
	AreaSqM     float64    `gorm:"not null;default:0" json:"area_sq_m"`
	Description string     `gorm:"size:512" json:"description"`
	Status      PlotStatus `gorm:"size:32;not null;default:active" json:"status"`
}

type CropBatch struct {
	model.Base
	PlotID          string      `gorm:"type:uuid;index;not null" json:"plot_id"`
	Plot            Plot        `gorm:"foreignKey:PlotID" json:"plot,omitempty"`
	ProductID       *string     `gorm:"type:uuid;index" json:"product_id"`
	CropName        string      `gorm:"size:128;not null" json:"crop_name"`
	PlantedAt       time.Time   `gorm:"not null" json:"planted_at"`
	ExpectedHarvest *time.Time  `json:"expected_harvest,omitempty"`
	Status          BatchStatus `gorm:"size:32;not null;default:planned" json:"status"`
	Note            string      `gorm:"size:1024" json:"note"`
	Activities      []CropActivity `gorm:"foreignKey:CropBatchID" json:"activities,omitempty"`
	Harvests        []Harvest   `gorm:"foreignKey:CropBatchID" json:"harvests,omitempty"`
}

type CropActivity struct {
	model.Base
	CropBatchID string       `gorm:"type:uuid;index;not null" json:"crop_batch_id"`
	Type        ActivityType `gorm:"size:32;not null" json:"type"`
	ActivityAt  time.Time    `gorm:"not null" json:"activity_at"`
	Note        string       `gorm:"size:512" json:"note"`
	CreatedBy   string       `gorm:"type:uuid" json:"created_by"`
}

type Harvest struct {
	model.Base
	CropBatchID string    `gorm:"type:uuid;index;not null" json:"crop_batch_id"`
	ProductID   string    `gorm:"type:uuid;index;not null" json:"product_id"`
	Quantity    float64   `gorm:"not null" json:"quantity"`
	HarvestedAt time.Time `gorm:"not null" json:"harvested_at"`
	Note        string    `gorm:"size:512" json:"note"`
	CreatedBy   string    `gorm:"type:uuid" json:"created_by"`
}
