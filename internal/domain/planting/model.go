package planting

import (
	"tinh-tien-api/internal/pkg/model"
)

// PlantingSchedule is the top-level document Flutter calls "planting-schedule".
type PlantingSchedule struct {
	model.Base
	VegetableName        string  `gorm:"size:128;not null" json:"vegetable_name"`
	SeedType             string  `gorm:"size:128" json:"seed_type"`
	PlantingDate         string  `gorm:"size:16" json:"planting_date"`           // YYYY-MM-DD
	ExpectedHarvestDate  string  `gorm:"size:16" json:"expected_harvest_date"`
	ActualHarvestDate    string  `gorm:"size:16" json:"actual_harvest_date"`
	Area                 string  `gorm:"size:32" json:"area"`
	SeedQuantity         string  `gorm:"size:32" json:"seed_quantity"`
	SeedUnit             string  `gorm:"size:32" json:"seed_unit"`
	Location             string  `gorm:"size:256" json:"location"`
	ExpectedYield        string  `gorm:"size:32" json:"expected_yield"`
	ActualYield          string  `gorm:"size:32" json:"actual_yield"`
	SeedCost             string  `gorm:"size:32" json:"seed_cost"`
	Notes                string  `gorm:"size:2048" json:"notes"`
	Status               int     `gorm:"not null;default:0" json:"status"`
	IsHarvested          bool    `gorm:"not null;default:false" json:"is_harvested"`

	SprayTasks     []SprayTask     `gorm:"foreignKey:ScheduleID;constraint:OnDelete:CASCADE" json:"spray_tasks,omitempty"`
	FertilizeTasks []FertilizeTask `gorm:"foreignKey:ScheduleID;constraint:OnDelete:CASCADE" json:"fertilize_tasks,omitempty"`
	PlantingTasks  []PlantingTask  `gorm:"foreignKey:ScheduleID;constraint:OnDelete:CASCADE" json:"planting_tasks,omitempty"`
}

type SprayTask struct {
	model.Base
	ScheduleID     string `gorm:"type:uuid;index;not null" json:"schedule_id"`
	SprayDate      string `gorm:"size:16;not null" json:"spray_date"`
	PesticideName  string `gorm:"size:128" json:"pesticide_name"`
	PesticideType  string `gorm:"size:128" json:"pesticide_type"`
	Dosage         string `gorm:"size:64" json:"dosage"`
	QuarantineDate string `gorm:"size:16" json:"quarantine_date"`
	QuarantineDays *int   `gorm:"" json:"quarantine_days"`
	Cost           string `gorm:"size:32" json:"cost"`
	Description    string `gorm:"size:1024" json:"description"`
}

type FertilizeTask struct {
	model.Base
	ScheduleID        string `gorm:"type:uuid;index;not null" json:"schedule_id"`
	FertilizeDate     string `gorm:"size:16;not null" json:"fertilize_date"`
	FertilizerName    string `gorm:"size:128" json:"fertilizer_name"`
	FertilizerType    string `gorm:"size:128" json:"fertilizer_type"`
	Amount            string `gorm:"size:32" json:"amount"`
	Unit              string `gorm:"size:32" json:"unit"`
	ApplicationMethod string `gorm:"size:128" json:"application_method"`
	ApplicationNumber *int   `gorm:"" json:"application_number"`
	Cost              string `gorm:"size:32" json:"cost"`
	Description       string `gorm:"size:1024" json:"description"`
}

type PlantingTask struct {
	model.Base
	ScheduleID  string `gorm:"type:uuid;index;not null" json:"schedule_id"`
	TaskName    string `gorm:"size:128;not null" json:"task_name"`
	TaskDate    string `gorm:"size:16;not null" json:"task_date"`
	TaskType    string `gorm:"size:64" json:"task_type"`
	Status      int    `gorm:"not null;default:0" json:"status"`
	Cost        string `gorm:"size:32" json:"cost"`
	Description string `gorm:"size:1024" json:"description"`
}
