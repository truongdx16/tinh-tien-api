package expense

type CreateExpenseRequest struct {
	Category    Category `json:"category"`
	Amount      int64    `json:"amount"`
	Description string   `json:"description"`
	ExpenseDate string   `json:"expense_date"`
}

type UpdateExpenseRequest struct {
	Category    *Category `json:"category"`
	Amount      *int64    `json:"amount"`
	Description *string   `json:"description"`
	ExpenseDate *string   `json:"expense_date"`
}

type ListQuery struct {
	From     string
	To       string
	Category Category
	Page     int
	PageSize int
}
