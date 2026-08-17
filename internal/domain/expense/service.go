package expense

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("expense not found")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(e *Expense) error { return r.db.Create(e).Error }
func (r *Repository) Update(e *Expense) error { return r.db.Save(e).Error }
func (r *Repository) Delete(id string) error { return r.db.Delete(&Expense{}, "id = ?", id).Error }

func (r *Repository) Get(id string) (*Expense, error) {
	var e Expense
	err := r.db.First(&e, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &e, nil
}

func (r *Repository) List(q ListQuery) ([]Expense, int64, error) {
	query := r.db.Model(&Expense{})
	if q.From != "" {
		query = query.Where("expense_date >= ?", q.From)
	}
	if q.To != "" {
		query = query.Where("expense_date <= ?", q.To)
	}
	if q.Category != "" {
		query = query.Where("category = ?", q.Category)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Expense
	offset := (q.Page - 1) * q.PageSize
	limit := q.PageSize
	if limit <= 0 {
		limit = 20
	}
	err := query.Order("expense_date desc").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func (r *Repository) SumByPeriod(from, to string) (int64, error) {
	query := r.db.Model(&Expense{})
	if from != "" {
		query = query.Where("expense_date >= ?", from)
	}
	if to != "" {
		query = query.Where("expense_date <= ?", to)
	}
	var total int64
	err := query.Select("COALESCE(SUM(amount), 0)").Scan(&total).Error
	return total, err
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func parseDate(s string) (time.Time, error) {
	layouts := []string{time.RFC3339, "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("invalid date format")
}

func (s *Service) Create(req CreateExpenseRequest, userID string) (*Expense, error) {
	date := time.Now()
	if req.ExpenseDate != "" {
		d, err := parseDate(req.ExpenseDate)
		if err != nil {
			return nil, err
		}
		date = d
	}
	cat := req.Category
	if cat == "" {
		cat = CategoryOther
	}
	e := &Expense{
		Category: cat, Amount: req.Amount, Description: req.Description,
		ExpenseDate: date, CreatedBy: userID,
	}
	return e, s.repo.Create(e)
}

func (s *Service) Update(id string, req UpdateExpenseRequest) (*Expense, error) {
	e, err := s.repo.Get(id)
	if err != nil {
		return nil, err
	}
	if req.Category != nil {
		e.Category = *req.Category
	}
	if req.Amount != nil {
		e.Amount = *req.Amount
	}
	if req.Description != nil {
		e.Description = *req.Description
	}
	if req.ExpenseDate != nil {
		d, err := parseDate(*req.ExpenseDate)
		if err != nil {
			return nil, err
		}
		e.ExpenseDate = d
	}
	return e, s.repo.Update(e)
}

func (s *Service) Delete(id string) error { return s.repo.Delete(id) }
func (s *Service) Get(id string) (*Expense, error) { return s.repo.Get(id) }
func (s *Service) List(q ListQuery) ([]Expense, int64, error) { return s.repo.List(q) }
func (s *Service) SumByPeriod(from, to string) (int64, error) { return s.repo.SumByPeriod(from, to) }
