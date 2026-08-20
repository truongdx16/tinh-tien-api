package planting

// WriteRequest is the body for create/update planting schedule.
type WriteRequest struct {
	VegetableName       string          `json:"vegetable_name"`
	SeedType            *string         `json:"seed_type"`
	PlantingDate        *string         `json:"planting_date"`
	ExpectedHarvestDate *string         `json:"expected_harvest_date"`
	ActualHarvestDate   *string         `json:"actual_harvest_date"`
	Area                *string         `json:"area"`
	SeedQuantity        *string         `json:"seed_quantity"`
	SeedUnit            *string         `json:"seed_unit"`
	Location            *string         `json:"location"`
	ExpectedYield       *string         `json:"expected_yield"`
	ActualYield         *string         `json:"actual_yield"`
	SeedCost            *string         `json:"seed_cost"`
	Notes               *string         `json:"notes"`
	Status              int             `json:"status"`
	IsHarvested         bool            `json:"is_harvested"`
	SprayTasks          []SprayTask     `json:"spray_tasks"`
	FertilizeTasks      []FertilizeTask `json:"fertilize_tasks"`
	PlantingTasks       []PlantingTask  `json:"planting_tasks"`
}
