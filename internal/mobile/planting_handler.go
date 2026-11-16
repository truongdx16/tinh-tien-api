package mobile

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"tinh-tien-api/internal/domain/planting"
	"tinh-tien-api/internal/mobile/adapter"
	"tinh-tien-api/internal/pkg/httputil"
)

type PlantingMobileHandler struct {
	svc *planting.Service
	loc *time.Location
}

func NewPlantingMobileHandler(svc *planting.Service, loc *time.Location) *PlantingMobileHandler {
	return &PlantingMobileHandler{svc: svc, loc: loc}
}

// ---- Response DTOs: empty values are null (Flutter turns "" dates into DateTime.now()) ----

type MobileSprayTaskDto struct {
	SprayDate      *string `json:"spray_date"`
	PesticideName  *string `json:"pesticide_name"`
	PesticideType  *string `json:"pesticide_type"`
	Dosage         *string `json:"dosage"`
	QuarantineDate *string `json:"quarantine_date"`
	QuarantineDays *int    `json:"quarantine_days"`
	Cost           *string `json:"cost"`
	Description    *string `json:"description"`
}

type MobileFertilizeTaskDto struct {
	FertilizeDate     *string `json:"fertilize_date"`
	FertilizerName    *string `json:"fertilizer_name"`
	FertilizerType    *string `json:"fertilizer_type"`
	Amount            *string `json:"amount"`
	Unit              *string `json:"unit"`
	ApplicationMethod *string `json:"application_method"`
	ApplicationNumber *int    `json:"application_number"`
	Cost              *string `json:"cost"`
	Description       *string `json:"description"`
}

type MobilePlantingTaskDto struct {
	TaskName    string  `json:"task_name"`
	TaskDate    *string `json:"task_date"`
	TaskType    *string `json:"task_type"`
	Status      int     `json:"status"`
	Cost        *string `json:"cost"`
	Description *string `json:"description"`
}

type MobilePlantingScheduleDto struct {
	ID                  string                   `json:"id"`
	VegetableName       string                   `json:"vegetable_name"`
	SeedType            *string                  `json:"seed_type"`
	PlantingDate        *string                  `json:"planting_date"`
	ExpectedHarvestDate *string                  `json:"expected_harvest_date"`
	ActualHarvestDate   *string                  `json:"actual_harvest_date"`
	Area                *string                  `json:"area"`
	SeedQuantity        *string                  `json:"seed_quantity"`
	SeedUnit            *string                  `json:"seed_unit"`
	Location            *string                  `json:"location"`
	ExpectedYield       *string                  `json:"expected_yield"`
	ActualYield         *string                  `json:"actual_yield"`
	SeedCost            *string                  `json:"seed_cost"`
	Notes               *string                  `json:"notes"`
	Status              int                      `json:"status"`
	IsHarvested         bool                     `json:"is_harvested"`
	SprayTasks          []MobileSprayTaskDto     `json:"spray_tasks"`
	FertilizeTasks      []MobileFertilizeTaskDto `json:"fertilize_tasks"`
	PlantingTasks       []MobilePlantingTaskDto  `json:"planting_tasks"`
	CreatedAt           string                   `json:"created_at"`
	UpdatedAt           string                   `json:"updated_at"`
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (h *PlantingMobileHandler) toDto(p planting.PlantingSchedule) MobilePlantingScheduleDto {
	dto := MobilePlantingScheduleDto{
		ID:                  p.ID.String(),
		VegetableName:       p.VegetableName,
		SeedType:            nullable(p.SeedType),
		PlantingDate:        nullable(p.PlantingDate),
		ExpectedHarvestDate: nullable(p.ExpectedHarvestDate),
		ActualHarvestDate:   nullable(p.ActualHarvestDate),
		Area:                nullable(p.Area),
		SeedQuantity:        nullable(p.SeedQuantity),
		SeedUnit:            nullable(p.SeedUnit),
		Location:            nullable(p.Location),
		ExpectedYield:       nullable(p.ExpectedYield),
		ActualYield:         nullable(p.ActualYield),
		SeedCost:            nullable(p.SeedCost),
		Notes:               nullable(p.Notes),
		Status:              p.Status,
		IsHarvested:         p.IsHarvested,
		SprayTasks:          make([]MobileSprayTaskDto, 0, len(p.SprayTasks)),
		FertilizeTasks:      make([]MobileFertilizeTaskDto, 0, len(p.FertilizeTasks)),
		PlantingTasks:       make([]MobilePlantingTaskDto, 0, len(p.PlantingTasks)),
		CreatedAt:           p.CreatedAt.In(h.loc).Format("2006-01-02"),
		UpdatedAt:           p.UpdatedAt.In(h.loc).Format("2006-01-02"),
	}
	for _, t := range p.SprayTasks {
		dto.SprayTasks = append(dto.SprayTasks, MobileSprayTaskDto{
			SprayDate: nullable(t.SprayDate), PesticideName: nullable(t.PesticideName),
			PesticideType: nullable(t.PesticideType), Dosage: nullable(t.Dosage),
			QuarantineDate: nullable(t.QuarantineDate), QuarantineDays: t.QuarantineDays,
			Cost: nullable(t.Cost), Description: nullable(t.Description),
		})
	}
	for _, t := range p.FertilizeTasks {
		dto.FertilizeTasks = append(dto.FertilizeTasks, MobileFertilizeTaskDto{
			FertilizeDate: nullable(t.FertilizeDate), FertilizerName: nullable(t.FertilizerName),
			FertilizerType: nullable(t.FertilizerType), Amount: nullable(t.Amount), Unit: nullable(t.Unit),
			ApplicationMethod: nullable(t.ApplicationMethod), ApplicationNumber: t.ApplicationNumber,
			Cost: nullable(t.Cost), Description: nullable(t.Description),
		})
	}
	for _, t := range p.PlantingTasks {
		dto.PlantingTasks = append(dto.PlantingTasks, MobilePlantingTaskDto{
			TaskName: t.TaskName, TaskDate: nullable(t.TaskDate), TaskType: nullable(t.TaskType),
			Status: t.Status, Cost: nullable(t.Cost), Description: nullable(t.Description),
		})
	}
	return dto
}

func (h *PlantingMobileHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List()
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "Không tải được lịch gieo trồng")
		return
	}
	dtos := make([]MobilePlantingScheduleDto, 0, len(items))
	for _, it := range items {
		dtos = append(dtos, h.toDto(it))
	}
	adapter.OK(w, "planting schedules retrieved", dtos)
}

func (h *PlantingMobileHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	item, err := h.svc.Get(id)
	if err != nil {
		if errors.Is(err, planting.ErrNotFound) {
			adapter.Fail(w, http.StatusNotFound, "Không tìm thấy lịch gieo trồng")
			return
		}
		adapter.Fail(w, http.StatusInternalServerError, "Không tải được lịch gieo trồng")
		return
	}
	adapter.OK(w, "planting schedule retrieved", h.toDto(*item))
}

func (h *PlantingMobileHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req planting.WriteRequest
	if err := httputil.Decode(r, &req); err != nil {
		adapter.Fail(w, http.StatusBadRequest, "Dữ liệu gửi lên không hợp lệ")
		return
	}
	item, err := h.svc.Create(req)
	if err != nil {
		adapter.Fail(w, http.StatusBadRequest, userMessage(err, "Không tạo được lịch gieo trồng"))
		return
	}
	adapter.Created(w, "planting schedule created", h.toDto(*item))
}

func (h *PlantingMobileHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req planting.WriteRequest
	if err := httputil.Decode(r, &req); err != nil {
		adapter.Fail(w, http.StatusBadRequest, "Dữ liệu gửi lên không hợp lệ")
		return
	}
	item, err := h.svc.Update(id, req)
	if err != nil {
		if errors.Is(err, planting.ErrNotFound) {
			adapter.Fail(w, http.StatusNotFound, "Không tìm thấy lịch gieo trồng")
			return
		}
		adapter.Fail(w, http.StatusInternalServerError, "Không cập nhật được lịch gieo trồng")
		return
	}
	adapter.OK(w, "planting schedule updated", h.toDto(*item))
}

func (h *PlantingMobileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(id); err != nil {
		if errors.Is(err, planting.ErrNotFound) {
			adapter.Fail(w, http.StatusNotFound, "Không tìm thấy lịch gieo trồng")
			return
		}
		adapter.Fail(w, http.StatusInternalServerError, "Không xoá được lịch gieo trồng")
		return
	}
	adapter.OK(w, "planting schedule deleted", nil)
}
