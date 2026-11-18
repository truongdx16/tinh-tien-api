package mobile

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"tinh-tien-api/internal/domain/customer"
	"tinh-tien-api/internal/domain/order"
	"tinh-tien-api/internal/domain/report"
)

type customerFixture struct {
	db      *gorm.DB
	router  http.Handler
	walkIn  customer.Customer
	regular customer.Customer
}

func newCustomerFixture(t *testing.T) customerFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&customer.Customer{}, &order.Order{}, &order.OrderItem{}, &order.Payment{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	customerSvc := customer.NewService(customer.NewRepository(db))
	walkIn, err := customerSvc.Create(customer.CreateCustomerRequest{Code: customer.WalkInCode, Name: "Khách vãng lai", IsWalkIn: true})
	if err != nil {
		t.Fatal(err)
	}
	regular, err := customerSvc.Create(customer.CreateCustomerRequest{Name: "Nguyễn Văn Thi", Phone: "0912345678"})
	if err != nil {
		t.Fatal(err)
	}

	orderSvc := order.NewService(order.NewRepository(db), nil, nil)
	reportSvc := report.NewService(report.NewRepository(db), nil, nil, orderSvc)
	loc := time.UTC

	customers := NewCustomerMobileHandler(customerSvc)
	orders := NewOrderMobileHandler(orderSvc, customerSvc, loc)
	stats := NewStatsMobileHandler(reportSvc, customerSvc, loc)

	r := chi.NewRouter()
	r.Get("/customers", customers.List)
	r.Delete("/customers/{id}", customers.Delete)
	r.Get("/orders", orders.List)
	r.Get("/stats/customers/{id}", stats.CustomerSummary)

	return customerFixture{db: db, router: r, walkIn: *walkIn, regular: *regular}
}

func (f customerFixture) addOrder(t *testing.T, customerID string, status order.Status, total, due int64, at time.Time) {
	t.Helper()
	o := order.Order{CustomerID: &customerID, Status: status, Subtotal: total, Total: total, BalanceDue: due, PaidAmount: total - due}
	if err := f.db.Create(&o).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Model(&o).UpdateColumn("created_at", at).Error; err != nil {
		t.Fatal(err)
	}
}

func (f customerFixture) do(t *testing.T, method, path string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	var body map[string]any
	_ = json.NewDecoder(w.Body).Decode(&body)
	return w.Code, body
}

func TestDeleteCustomer(t *testing.T) {
	f := newCustomerFixture(t)
	f.addOrder(t, f.regular.ID.String(), order.StatusDelivered, 50000, 0, time.Now())

	if code, _ := f.do(t, http.MethodDelete, "/customers/"+customer.WalkInCode); code != http.StatusConflict {
		t.Fatalf("deleting walk-in: status %d, want 409", code)
	}
	if code, _ := f.do(t, http.MethodDelete, "/customers/"+f.regular.ID.String()); code != http.StatusOK {
		t.Fatalf("delete: status %d, want 200", code)
	}
	if code, _ := f.do(t, http.MethodDelete, "/customers/"+f.regular.ID.String()); code != http.StatusNotFound {
		t.Fatalf("delete twice: status %d, want 404", code)
	}

	// Gone from the customer list...
	_, list := f.do(t, http.MethodGet, "/customers?per_page=50")
	for _, c := range list["data"].(map[string]any)["data"].([]any) {
		if c.(map[string]any)["name"] == "Nguyễn Văn Thi" {
			t.Fatal("deleted customer still listed")
		}
	}
	// ...but their old order still shows who bought.
	_, orders := f.do(t, http.MethodGet, "/orders")
	got := orders["data"].([]any)
	if len(got) != 1 {
		t.Fatalf("orders = %d, want 1", len(got))
	}
	cust, _ := got[0].(map[string]any)["customer"].(map[string]any)
	if cust == nil || cust["name"] != "Nguyễn Văn Thi" {
		t.Fatalf("old order lost its customer: %v", got[0])
	}
}

func TestCustomerSummaryAndOrderFilter(t *testing.T) {
	f := newCustomerFixture(t)
	id := f.regular.ID.String()
	day := func(d int) time.Time { return time.Date(2026, 11, d, 9, 0, 0, 0, time.UTC) }
	f.addOrder(t, id, order.StatusDelivered, 100000, 0, day(3))
	f.addOrder(t, id, order.StatusConfirmed, 60000, 20000, day(10))
	f.addOrder(t, id, order.StatusCancelled, 999000, 0, day(11)) // not counted
	f.addOrder(t, f.walkIn.ID.String(), order.StatusDelivered, 5000, 0, day(10))

	code, body := f.do(t, http.MethodGet, "/stats/customers/"+id)
	if code != http.StatusOK {
		t.Fatalf("summary status %d: %v", code, body)
	}
	s := body["data"].(map[string]any)
	want := map[string]any{
		"order_count":   float64(2),
		"revenue":       "160000",
		"paid":          "140000",
		"debt":          "20000",
		"average_order": "80000",
	}
	for k, v := range want {
		if s[k] != v {
			t.Errorf("%s = %v, want %v", k, s[k], v)
		}
	}
	if s["first_order_at"] == nil || s["last_order_at"] == nil {
		t.Errorf("first/last order dates missing: %v", s)
	}

	// Date range: only the order of the 10th.
	_, body = f.do(t, http.MethodGet, "/stats/customers/"+id+"?from=2026-11-05&to=2026-11-30")
	if got := body["data"].(map[string]any)["revenue"]; got != "60000" {
		t.Errorf("ranged revenue = %v, want 60000", got)
	}

	// No orders: zeros, no dates.
	other := customer.Customer{Name: "Mới"}
	f.db.Create(&other)
	_, body = f.do(t, http.MethodGet, "/stats/customers/"+other.ID.String())
	if s := body["data"].(map[string]any); s["order_count"] != float64(0) || s["last_order_at"] != nil {
		t.Errorf("empty summary = %v", s)
	}

	// Orders of one customer, walk-in code accepted.
	_, body = f.do(t, http.MethodGet, "/orders?customer_id="+id)
	if n := len(body["data"].([]any)); n != 3 {
		t.Errorf("orders for customer = %d, want 3", n)
	}
	_, body = f.do(t, http.MethodGet, "/orders?customer_id="+customer.WalkInCode)
	if n := len(body["data"].([]any)); n != 1 {
		t.Errorf("orders for walk-in = %d, want 1", n)
	}
	_, body = f.do(t, http.MethodGet, "/orders?customer_id="+id+"&limit=1")
	if n := len(body["data"].([]any)); n != 1 {
		t.Errorf("limit=1 returned %d orders", n)
	}
}
