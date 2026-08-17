package crop

import (
	"errors"
	"time"

	"tinh-tien-api/internal/domain/inventory"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("not found")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreatePlot(p *Plot) error { return r.db.Create(p).Error }
func (r *Repository) UpdatePlot(p *Plot) error { return r.db.Save(p).Error }
func (r *Repository) DeletePlot(id string) error { return r.db.Delete(&Plot{}, "id = ?", id).Error }

func (r *Repository) GetPlot(id string) (*Plot, error) {
	var p Plot
	err := r.db.First(&p, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *Repository) ListPlots(page, pageSize int) ([]Plot, int64, error) {
	var total int64
	if err := r.db.Model(&Plot{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Plot
	offset := (page - 1) * pageSize
	err := r.db.Order("name asc").Offset(offset).Limit(pageSize).Find(&items).Error
	return items, total, err
}

func (r *Repository) CreateBatch(b *CropBatch) error { return r.db.Create(b).Error }
func (r *Repository) UpdateBatch(b *CropBatch) error { return r.db.Save(b).Error }

func (r *Repository) GetBatch(id string) (*CropBatch, error) {
	var b CropBatch
	err := r.db.Preload("Plot").Preload("Activities").Preload("Harvests").First(&b, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &b, nil
}

func (r *Repository) ListBatches(q BatchListQuery) ([]CropBatch, int64, error) {
	query := r.db.Model(&CropBatch{})
	if q.PlotID != "" {
		query = query.Where("plot_id = ?", q.PlotID)
	}
	if q.Status != "" {
		query = query.Where("status = ?", q.Status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []CropBatch
	offset := (q.Page - 1) * q.PageSize
	limit := q.PageSize
	if limit <= 0 {
		limit = 20
	}
	err := query.Preload("Plot").Order("planted_at desc").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func (r *Repository) CreateActivity(a *CropActivity) error { return r.db.Create(a).Error }
func (r *Repository) CreateHarvest(h *Harvest) error { return r.db.Create(h).Error }

func (r *Repository) ListDueHarvests(withinDays, page, pageSize int) ([]CropBatch, int64, error) {
	deadline := time.Now().AddDate(0, 0, withinDays)
	base := r.db.Model(&CropBatch{}).
		Where("expected_harvest IS NOT NULL AND expected_harvest <= ? AND status IN ?", deadline, []BatchStatus{BatchPlanned, BatchGrowing})
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []CropBatch
	offset := (page - 1) * pageSize
	err := r.db.Preload("Plot").
		Where("expected_harvest IS NOT NULL AND expected_harvest <= ? AND status IN ?", deadline, []BatchStatus{BatchPlanned, BatchGrowing}).
		Order("expected_harvest asc").
		Offset(offset).Limit(pageSize).
		Find(&items).Error
	return items, total, err
}

type Service struct {
	repo   *Repository
	invSvc *inventory.Service
}

func NewService(repo *Repository, invSvc *inventory.Service) *Service {
	return &Service{repo: repo, invSvc: invSvc}
}

func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, errors.New("date required")
	}
	layouts := []string{time.RFC3339, "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("invalid date format")
}

func (s *Service) CreatePlot(req CreatePlotRequest) (*Plot, error) {
	p := &Plot{
		Name: req.Name, Location: req.Location, AreaSqM: req.AreaSqM,
		Description: req.Description, Status: PlotActive,
	}
	return p, s.repo.CreatePlot(p)
}

func (s *Service) UpdatePlot(id string, req UpdatePlotRequest) (*Plot, error) {
	p, err := s.repo.GetPlot(id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Location != nil {
		p.Location = *req.Location
	}
	if req.AreaSqM != nil {
		p.AreaSqM = *req.AreaSqM
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if req.Status != nil {
		p.Status = *req.Status
	}
	return p, s.repo.UpdatePlot(p)
}

func (s *Service) DeletePlot(id string) error { return s.repo.DeletePlot(id) }
func (s *Service) GetPlot(id string) (*Plot, error) { return s.repo.GetPlot(id) }
func (s *Service) ListPlots(page, pageSize int) ([]Plot, int64, error) { return s.repo.ListPlots(page, pageSize) }

func (s *Service) CreateBatch(req CreateBatchRequest) (*CropBatch, error) {
	plantedAt, err := parseDate(req.PlantedAt)
	if err != nil {
		return nil, err
	}
	b := &CropBatch{
		PlotID: req.PlotID, ProductID: req.ProductID, CropName: req.CropName,
		PlantedAt: plantedAt, Status: BatchPlanned, Note: req.Note,
	}
	if req.ExpectedHarvest != nil && *req.ExpectedHarvest != "" {
		t, err := parseDate(*req.ExpectedHarvest)
		if err != nil {
			return nil, err
		}
		b.ExpectedHarvest = &t
	}
	return b, s.repo.CreateBatch(b)
}

func (s *Service) UpdateBatch(id string, req UpdateBatchRequest) (*CropBatch, error) {
	b, err := s.repo.GetBatch(id)
	if err != nil {
		return nil, err
	}
	if req.CropName != nil {
		b.CropName = *req.CropName
	}
	if req.Note != nil {
		b.Note = *req.Note
	}
	if req.Status != nil {
		b.Status = *req.Status
	}
	if req.ExpectedHarvest != nil {
		if *req.ExpectedHarvest == "" {
			b.ExpectedHarvest = nil
		} else {
			t, err := parseDate(*req.ExpectedHarvest)
			if err != nil {
				return nil, err
			}
			b.ExpectedHarvest = &t
		}
	}
	return b, s.repo.UpdateBatch(b)
}

func (s *Service) GetBatch(id string) (*CropBatch, error) { return s.repo.GetBatch(id) }
func (s *Service) ListBatches(q BatchListQuery) ([]CropBatch, int64, error) { return s.repo.ListBatches(q) }

func (s *Service) AddActivity(batchID string, req CreateActivityRequest, userID string) (*CropActivity, error) {
	at, err := parseDate(req.ActivityAt)
	if err != nil {
		return nil, err
	}
	a := &CropActivity{
		CropBatchID: batchID, Type: req.Type, ActivityAt: at, Note: req.Note, CreatedBy: userID,
	}
	return a, s.repo.CreateActivity(a)
}

func (s *Service) RecordHarvest(batchID string, req CreateHarvestRequest, userID string) (*Harvest, error) {
	harvestedAt, err := parseDate(req.HarvestedAt)
	if err != nil {
		return nil, err
	}
	if req.ProductID == "" || req.Quantity <= 0 {
		return nil, errors.New("product_id and quantity required")
	}

	h := &Harvest{
		CropBatchID: batchID, ProductID: req.ProductID, Quantity: req.Quantity,
		HarvestedAt: harvestedAt, Note: req.Note, CreatedBy: userID,
	}
	if err := s.repo.CreateHarvest(h); err != nil {
		return nil, err
	}

	ref := h.ID.String()
	m := &inventory.Movement{
		ProductID: req.ProductID, Type: inventory.MovementHarvest,
		Quantity: req.Quantity, ReferenceID: &ref, Note: "crop harvest", CreatedBy: userID,
	}
	if err := s.invSvc.RecordMovement(m); err != nil {
		return nil, err
	}

	batch, err := s.repo.GetBatch(batchID)
	if err == nil {
		batch.Status = BatchHarvested
		_ = s.repo.UpdateBatch(batch)
	}

	return h, nil
}

func (s *Service) ListDueHarvests(withinDays, page, pageSize int) ([]DueHarvestAlert, int64, error) {
	if withinDays <= 0 {
		withinDays = 7
	}
	batches, total, err := s.repo.ListDueHarvests(withinDays, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	now := time.Now()
	items := make([]DueHarvestAlert, 0, len(batches))
	for _, b := range batches {
		if b.ExpectedHarvest == nil {
			continue
		}
		days := int(b.ExpectedHarvest.Sub(now).Hours() / 24)
		plotName := ""
		if b.Plot.ID.String() != "" {
			plotName = b.Plot.Name
		}
		items = append(items, DueHarvestAlert{
			BatchID: b.ID.String(), CropName: b.CropName, PlotName: plotName,
			ExpectedHarvest: b.ExpectedHarvest.Format("2006-01-02"), DaysUntil: days,
		})
	}
	return items, total, nil
}
