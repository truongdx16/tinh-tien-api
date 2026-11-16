package planting

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// FlexString accepts a JSON string, number or null and stores it as a string.
// Flutter sends area/cost/yield as numbers (double) and dates as "YYYY-MM-DD".
type FlexString string

func (f *FlexString) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		*f = ""
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = FlexString(strings.TrimSpace(s))
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	// Normalise "12.0" → "12" so values round-trip cleanly.
	if fl, err := n.Float64(); err == nil {
		*f = FlexString(strconv.FormatFloat(fl, 'f', -1, 64))
		return nil
	}
	*f = FlexString(n.String())
	return nil
}

// FlexInt accepts a JSON number, numeric string or null.
type FlexInt struct{ V *int }

func (f *FlexInt) UnmarshalJSON(b []byte) error {
	var s FlexString
	if err := s.UnmarshalJSON(b); err != nil {
		return err
	}
	f.V = nil
	if s == "" {
		return nil
	}
	if fl, err := strconv.ParseFloat(string(s), 64); err == nil {
		v := int(fl)
		f.V = &v
	}
	return nil
}

// dateOnly trims a timestamp to "YYYY-MM-DD" (Flutter TimestampSerializer format).
func dateOnly(s FlexString) string {
	v := string(s)
	if len(v) >= 10 {
		return v[:10]
	}
	return v
}

type SprayTaskInput struct {
	SprayDate      FlexString `json:"spray_date"`
	PesticideName  FlexString `json:"pesticide_name"`
	PesticideType  FlexString `json:"pesticide_type"`
	Dosage         FlexString `json:"dosage"`
	QuarantineDate FlexString `json:"quarantine_date"`
	QuarantineDays FlexInt    `json:"quarantine_days"`
	Cost           FlexString `json:"cost"`
	Description    FlexString `json:"description"`
}

type FertilizeTaskInput struct {
	FertilizeDate     FlexString `json:"fertilize_date"`
	FertilizerName    FlexString `json:"fertilizer_name"`
	FertilizerType    FlexString `json:"fertilizer_type"`
	Amount            FlexString `json:"amount"`
	Unit              FlexString `json:"unit"`
	ApplicationMethod FlexString `json:"application_method"`
	ApplicationNumber FlexInt    `json:"application_number"`
	Cost              FlexString `json:"cost"`
	Description       FlexString `json:"description"`
}

type PlantingTaskInput struct {
	TaskName    FlexString `json:"task_name"`
	TaskDate    FlexString `json:"task_date"`
	TaskType    FlexString `json:"task_type"`
	Status      FlexInt    `json:"status"`
	Cost        FlexString `json:"cost"`
	Description FlexString `json:"description"`
}

// WriteRequest is the body for create/update planting schedule (Flutter PlantingSchedule.toJson).
type WriteRequest struct {
	VegetableName       FlexString           `json:"vegetable_name"`
	SeedType            FlexString           `json:"seed_type"`
	PlantingDate        FlexString           `json:"planting_date"`
	ExpectedHarvestDate FlexString           `json:"expected_harvest_date"`
	ActualHarvestDate   FlexString           `json:"actual_harvest_date"`
	Area                FlexString           `json:"area"`
	SeedQuantity        FlexString           `json:"seed_quantity"`
	SeedUnit            FlexString           `json:"seed_unit"`
	Location            FlexString           `json:"location"`
	ExpectedYield       FlexString           `json:"expected_yield"`
	ActualYield         FlexString           `json:"actual_yield"`
	SeedCost            FlexString           `json:"seed_cost"`
	Notes               FlexString           `json:"notes"`
	Status              FlexInt              `json:"status"`
	IsHarvested         bool                 `json:"is_harvested"`
	SprayTasks          []SprayTaskInput     `json:"spray_tasks"`
	FertilizeTasks      []FertilizeTaskInput `json:"fertilize_tasks"`
	PlantingTasks       []PlantingTaskInput  `json:"planting_tasks"`
}

func (in SprayTaskInput) toModel() SprayTask {
	return SprayTask{
		SprayDate:      dateOnly(in.SprayDate),
		PesticideName:  string(in.PesticideName),
		PesticideType:  string(in.PesticideType),
		Dosage:         string(in.Dosage),
		QuarantineDate: dateOnly(in.QuarantineDate),
		QuarantineDays: in.QuarantineDays.V,
		Cost:           string(in.Cost),
		Description:    string(in.Description),
	}
}

func (in FertilizeTaskInput) toModel() FertilizeTask {
	return FertilizeTask{
		FertilizeDate:     dateOnly(in.FertilizeDate),
		FertilizerName:    string(in.FertilizerName),
		FertilizerType:    string(in.FertilizerType),
		Amount:            string(in.Amount),
		Unit:              string(in.Unit),
		ApplicationMethod: string(in.ApplicationMethod),
		ApplicationNumber: in.ApplicationNumber.V,
		Cost:              string(in.Cost),
		Description:       string(in.Description),
	}
}

func (in PlantingTaskInput) toModel() PlantingTask {
	status := 0
	if in.Status.V != nil {
		status = *in.Status.V
	}
	return PlantingTask{
		TaskName:    string(in.TaskName),
		TaskDate:    dateOnly(in.TaskDate),
		TaskType:    string(in.TaskType),
		Status:      status,
		Cost:        string(in.Cost),
		Description: string(in.Description),
	}
}
