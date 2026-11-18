package mobile

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"tinh-tien-api/internal/domain/order"
	"tinh-tien-api/internal/domain/product"
)

type productFixture struct {
	router   http.Handler
	category product.Category
	unit     product.Unit
}

func newProductFixture(t *testing.T) productFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// order tables: product metrics read order_items.
	if err := db.AutoMigrate(&product.Category{}, &product.Unit{}, &product.Product{},
		&product.ProductCategory{}, &order.Order{}, &order.OrderItem{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	cat := product.Category{Name: "Rau", Active: true}
	unit := product.Unit{Name: "kg", Active: true}
	db.Create(&cat)
	db.Create(&unit)

	h := NewCatalogMobileHandler(product.NewService(product.NewRepository(db)))
	r := chi.NewRouter()
	r.Get("/products", h.ListProducts)
	r.Post("/products", h.CreateProduct)
	r.Put("/products/{id}", h.UpdateProduct)
	r.Delete("/products/{id}", h.DeleteProduct)
	return productFixture{router: r, category: cat, unit: unit}
}

func (f productFixture) send(t *testing.T, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	var out map[string]any
	_ = json.NewDecoder(w.Body).Decode(&out)
	return w.Code, out
}

func TestCreateProductValidates(t *testing.T) {
	f := newProductFixture(t)
	cases := []struct {
		body map[string]any
		msg  string
	}{
		{map[string]any{"name": "   ", "price": 1000, "status": 1}, "Tên sản phẩm không được để trống"},
		{map[string]any{"name": "Cà chua", "price": -5, "status": 1}, "Giá bán không được âm"},
		{map[string]any{"name": "Cà chua", "price": 1000, "status": 1, "category_ids": []string{"bad"}}, "Danh mục không hợp lệ"},
	}
	for _, c := range cases {
		code, body := f.send(t, http.MethodPost, "/products", c.body)
		if code != http.StatusBadRequest || body["message"] != c.msg {
			t.Errorf("%v → %d %v, want 400 %q", c.body, code, body["message"], c.msg)
		}
	}
}

func TestProductCreateUpdateDelete(t *testing.T) {
	f := newProductFixture(t)
	unitID, catID := f.unit.ID.String(), f.category.ID.String()

	code, body := f.send(t, http.MethodPost, "/products", map[string]any{
		"name": "  Cà chua bi ", "price": 45000, "sensitivity": 0.5, "status": 1,
		"unit_id": unitID, "category_ids": []string{catID},
		"image_url": "https://example.com/a.jpg",
	})
	if code != http.StatusCreated {
		t.Fatalf("create: %d %v", code, body)
	}
	p := body["data"].(map[string]any)
	id := p["id"].(string)
	if p["name"] != "Cà chua bi" {
		t.Errorf("name not trimmed: %q", p["name"])
	}
	if cats, _ := p["categories"].([]any); len(cats) != 1 {
		t.Errorf("categories = %v, want 1", p["categories"])
	}
	if u, _ := p["unit"].(map[string]any); u == nil || u["name"] != "kg" {
		t.Errorf("unit = %v, want kg", p["unit"])
	}

	code, body = f.send(t, http.MethodPut, "/products/"+id, map[string]any{
		"name": "Cà chua bi Đà Lạt", "price": 50000, "status": 0,
		"unit_id": unitID, "category_ids": []string{},
	})
	if code != http.StatusOK {
		t.Fatalf("update: %d %v", code, body)
	}
	p = body["data"].(map[string]any)
	if p["name"] != "Cà chua bi Đà Lạt" || p["status"] != float64(0) {
		t.Errorf("updated product = %v", p)
	}
	if cats, _ := p["categories"].([]any); len(cats) != 0 {
		t.Errorf("categories should be cleared, got %v", p["categories"])
	}

	if code, body = f.send(t, http.MethodPut, "/products/"+id, map[string]any{"name": "", "price": 1}); code != http.StatusBadRequest {
		t.Errorf("update with empty name: %d %v", code, body)
	}

	if code, _ = f.send(t, http.MethodDelete, "/products/"+id, nil); code != http.StatusOK {
		t.Fatalf("delete: %d", code)
	}
	if code, _ = f.send(t, http.MethodDelete, "/products/"+id, nil); code != http.StatusNotFound {
		t.Errorf("delete twice: %d, want 404", code)
	}
	if code, _ = f.send(t, http.MethodDelete, "/products/not-a-uuid", nil); code != http.StatusNotFound {
		t.Errorf("delete bad id: %d, want 404", code)
	}
	_, body = f.send(t, http.MethodGet, "/products?per_page=50", nil)
	if items := body["data"].(map[string]any)["data"].([]any); len(items) != 0 {
		t.Errorf("deleted product still listed: %v", items)
	}
}
