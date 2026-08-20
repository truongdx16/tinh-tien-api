package planting

import (
	"errors"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("planting schedule not found")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) preload(q *gorm.DB) *gorm.DB {
	return q.Preload("SprayTasks").Preload("FertilizeTasks").Preload("PlantingTasks")
}

func (r *Repository) List() ([]PlantingSchedule, error) {
	var items []PlantingSchedule
	err := r.preload(r.db).Order("created_at desc").Find(&items).Error
	return items, err
}

func (r *Repository) Get(id string) (*PlantingSchedule, error) {
	var s PlantingSchedule
	err := r.preload(r.db).First(&s, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *Repository) Create(s *PlantingSchedule) error {
	return r.db.Create(s).Error
}

func (r *Repository) Save(s *PlantingSchedule) error {
	return r.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(s).Error
}

func (r *Repository) Delete(id string) error {
	return r.db.Delete(&PlantingSchedule{}, "id = ?", id).Error
}

func (r *Repository) replaceNested(tx *gorm.DB, scheduleID string, s *PlantingSchedule) error {
	if err := tx.Where("schedule_id = ?", scheduleID).Delete(&SprayTask{}).Error; err != nil {
		return err
	}
	if err := tx.Where("schedule_id = ?", scheduleID).Delete(&FertilizeTask{}).Error; err != nil {
		return err
	}
	if err := tx.Where("schedule_id = ?", scheduleID).Delete(&PlantingTask{}).Error; err != nil {
		return err
	}
	for i := range s.SprayTasks {
		s.SprayTasks[i].ScheduleID = scheduleID
		if err := tx.Create(&s.SprayTasks[i]).Error; err != nil {
			return err
		}
	}
	for i := range s.FertilizeTasks {
		s.FertilizeTasks[i].ScheduleID = scheduleID
		if err := tx.Create(&s.FertilizeTasks[i]).Error; err != nil {
			return err
		}
	}
	for i := range s.PlantingTasks {
		s.PlantingTasks[i].ScheduleID = scheduleID
		if err := tx.Create(&s.PlantingTasks[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

// ---- Service ----

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List() ([]PlantingSchedule, error) {
	return s.repo.List()
}

func (s *Service) Get(id string) (*PlantingSchedule, error) {
	return s.repo.Get(id)
}

func (s *Service) Create(req WriteRequest) (*PlantingSchedule, error) {
	sched := reqToModel(req)
	if err := s.repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(sched).Error; err != nil {
			return err
		}
		return s.repo.replaceNested(tx, sched.ID.String(), sched)
	}); err != nil {
		return nil, err
	}
	return s.repo.Get(sched.ID.String())
}

func (s *Service) Update(id string, req WriteRequest) (*PlantingSchedule, error) {
	sched, err := s.repo.Get(id)
	if err != nil {
		return nil, err
	}
	applyReq(sched, req)
	if err := s.repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(sched).Error; err != nil {
			return err
		}
		return s.repo.replaceNested(tx, id, sched)
	}); err != nil {
		return nil, err
	}
	return s.repo.Get(id)
}

func (s *Service) Delete(id string) error {
	if _, err := s.repo.Get(id); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

func reqToModel(req WriteRequest) *PlantingSchedule {
	s := &PlantingSchedule{
		VegetableName:       req.VegetableName,
		SeedType:            strVal(req.SeedType),
		PlantingDate:        strVal(req.PlantingDate),
		ExpectedHarvestDate: strVal(req.ExpectedHarvestDate),
		ActualHarvestDate:   strVal(req.ActualHarvestDate),
		Area:                strVal(req.Area),
		SeedQuantity:        strVal(req.SeedQuantity),
		SeedUnit:            strVal(req.SeedUnit),
		Location:            strVal(req.Location),
		ExpectedYield:       strVal(req.ExpectedYield),
		ActualYield:         strVal(req.ActualYield),
		SeedCost:            strVal(req.SeedCost),
		Notes:               strVal(req.Notes),
		Status:              req.Status,
		IsHarvested:         req.IsHarvested,
		SprayTasks:          req.SprayTasks,
		FertilizeTasks:      req.FertilizeTasks,
		PlantingTasks:       req.PlantingTasks,
	}
	return s
}

func applyReq(s *PlantingSchedule, req WriteRequest) {
	s.VegetableName = req.VegetableName
	s.SeedType = strVal(req.SeedType)
	s.PlantingDate = strVal(req.PlantingDate)
	s.ExpectedHarvestDate = strVal(req.ExpectedHarvestDate)
	s.ActualHarvestDate = strVal(req.ActualHarvestDate)
	s.Area = strVal(req.Area)
	s.SeedQuantity = strVal(req.SeedQuantity)
	s.SeedUnit = strVal(req.SeedUnit)
	s.Location = strVal(req.Location)
	s.ExpectedYield = strVal(req.ExpectedYield)
	s.ActualYield = strVal(req.ActualYield)
	s.SeedCost = strVal(req.SeedCost)
	s.Notes = strVal(req.Notes)
	s.Status = req.Status
	s.IsHarvested = req.IsHarvested
	s.SprayTasks = req.SprayTasks
	s.FertilizeTasks = req.FertilizeTasks
	s.PlantingTasks = req.PlantingTasks
}

func strVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
