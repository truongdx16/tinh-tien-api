package crop

type CreatePlotRequest struct {
	Name        string  `json:"name"`
	Location    string  `json:"location"`
	AreaSqM     float64 `json:"area_sq_m"`
	Description string  `json:"description"`
}

type UpdatePlotRequest struct {
	Name        *string     `json:"name"`
	Location    *string     `json:"location"`
	AreaSqM     *float64    `json:"area_sq_m"`
	Description *string     `json:"description"`
	Status      *PlotStatus `json:"status"`
}

type CreateBatchRequest struct {
	PlotID          string  `json:"plot_id"`
	ProductID       *string `json:"product_id"`
	CropName        string  `json:"crop_name"`
	PlantedAt       string  `json:"planted_at"`
	ExpectedHarvest *string `json:"expected_harvest"`
	Note            string  `json:"note"`
}

type UpdateBatchRequest struct {
	CropName        *string      `json:"crop_name"`
	ExpectedHarvest *string      `json:"expected_harvest"`
	Status          *BatchStatus `json:"status"`
	Note            *string      `json:"note"`
}

type CreateActivityRequest struct {
	Type       ActivityType `json:"type"`
	ActivityAt string       `json:"activity_at"`
	Note       string       `json:"note"`
}

type CreateHarvestRequest struct {
	ProductID   string  `json:"product_id"`
	Quantity    float64 `json:"quantity"`
	HarvestedAt string  `json:"harvested_at"`
	Note        string  `json:"note"`
}

type BatchListQuery struct {
	PlotID   string
	Status   BatchStatus
	Page     int
	PageSize int
}

type DueHarvestAlert struct {
	BatchID         string  `json:"batch_id"`
	CropName        string  `json:"crop_name"`
	PlotName        string  `json:"plot_name"`
	ExpectedHarvest string  `json:"expected_harvest"`
	DaysUntil       int     `json:"days_until"`
}
