package mobile

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"tinh-tien-api/internal/domain/auth"
	"tinh-tien-api/internal/domain/customer"
	"tinh-tien-api/internal/domain/inventory"
	"tinh-tien-api/internal/domain/order"
	"tinh-tien-api/internal/domain/product"
)

func openTestDB(t *testing.T, models ...any) *gorm.DB {
	t.Helper()
	// One in-memory database shared by every connection of the pool (a plain
	// ":memory:" gives each connection its own empty database).
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// asUser runs the request as a signed-in user with [role].
func asUser(req *http.Request, userID string, role auth.Role) *http.Request {
	return req.WithContext(auth.WithContext(req.Context(), &auth.Claims{UserID: userID, Role: role}))
}

func TestDeleteOrder(t *testing.T) {
	db := openTestDB(t, &customer.Customer{}, &order.Order{}, &order.OrderItem{}, &order.Payment{},
		&inventory.Movement{}, &inventory.Balance{})
	invSvc := inventory.NewService(inventory.NewRepository(db))
	orderSvc := order.NewService(order.NewRepository(db), nil, invSvc)
	h := NewOrderMobileHandler(orderSvc, customer.NewService(customer.NewRepository(db)), time.UTC)
	r := chi.NewRouter()
	r.Get("/orders", h.List)
	r.Delete("/orders/{id}", h.Delete)

	productID := uuid.NewString()
	newOrder := func(status order.Status, movements ...inventory.MovementType) string {
		o := order.Order{Status: status, Total: 40000, Subtotal: 40000,
			Items: []order.OrderItem{{ProductID: productID, Name: "Rau", Unit: "kg", Quantity: 2}}}
		if err := db.Create(&o).Error; err != nil {
			t.Fatal(err)
		}
		ref := o.ID.String()
		for _, m := range movements {
			qty := -2.0
			if m == inventory.MovementReturn {
				qty = 2
			}
			if err := invSvc.RecordMovement(&inventory.Movement{ProductID: productID, Type: m, Quantity: qty, ReferenceID: &ref}); err != nil {
				t.Fatal(err)
			}
		}
		return ref
	}
	del := func(id string, role auth.Role) int {
		req := asUser(httptest.NewRequest(http.MethodDelete, "/orders/"+id, nil), "u1", role)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
	returns := func(id string) int64 {
		var n int64
		db.Model(&inventory.Movement{}).Where("reference_id = ? AND type = ?", id, inventory.MovementReturn).Count(&n)
		return n
	}

	confirmed := newOrder(order.StatusConfirmed, inventory.MovementSale)
	if code := del(confirmed, auth.RoleStaff); code != http.StatusForbidden {
		t.Fatalf("staff delete: %d, want 403", code)
	}
	if code := del(confirmed, auth.RoleOwner); code != http.StatusOK {
		t.Fatalf("owner delete: %d, want 200", code)
	}
	if n := returns(confirmed); n != 1 {
		t.Errorf("stock returned %d times, want 1", n)
	}
	if code := del(confirmed, auth.RoleOwner); code != http.StatusNotFound {
		t.Errorf("delete twice: %d, want 404", code)
	}
	var items int64
	db.Model(&order.OrderItem{}).Where("order_id = ?", confirmed).Count(&items)
	if items != 0 {
		t.Errorf("order items left: %d", items)
	}

	// Cancelled: stock was already returned, not again.
	cancelled := newOrder(order.StatusCancelled, inventory.MovementSale, inventory.MovementReturn)
	if code := del(cancelled, auth.RoleOwner); code != http.StatusOK {
		t.Fatalf("delete cancelled: %d", code)
	}
	if n := returns(cancelled); n != 1 {
		t.Errorf("cancelled order restocked again: %d returns", n)
	}

	// POS order without stock movements: nothing to return.
	pos := newOrder(order.StatusDelivered)
	if code := del(pos, auth.RoleOwner); code != http.StatusOK {
		t.Fatalf("delete pos order: %d", code)
	}
	if n := returns(pos); n != 0 {
		t.Errorf("pos order created %d returns", n)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orders", nil))
	var body map[string]any
	_ = json.NewDecoder(w.Body).Decode(&body)
	if n := len(body["data"].([]any)); n != 0 {
		t.Errorf("deleted orders still listed: %d", n)
	}
}

func TestChangePassword(t *testing.T) {
	db := openTestDB(t, &auth.User{})
	svc := auth.NewService(auth.NewRepository(db), "test-secret", time.Hour)
	user, err := svc.CreateUser(auth.CreateUserRequest{Username: "chu", Password: "old-pass", Role: auth.RoleStaff})
	if err != nil {
		t.Fatal(err)
	}
	h := NewAuthMobileHandler(svc)

	change := func(current, next string) (int, string) {
		b, _ := json.Marshal(map[string]string{"current_password": current, "new_password": next})
		req := asUser(httptest.NewRequest(http.MethodPost, "/auth/change-password", bytes.NewReader(b)),
			user.ID.String(), auth.RoleStaff)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ChangePassword(w, req)
		var body map[string]any
		_ = json.NewDecoder(w.Body).Decode(&body)
		msg, _ := body["message"].(string)
		return w.Code, msg
	}

	if code, msg := change("wrong", "new-pass-1"); code != http.StatusBadRequest || msg != "Mật khẩu hiện tại không đúng" {
		t.Errorf("wrong current: %d %q", code, msg)
	}
	if code, _ := change("old-pass", "123"); code != http.StatusBadRequest {
		t.Errorf("too short: %d, want 400", code)
	}
	if code, _ := change("old-pass", "old-pass"); code != http.StatusBadRequest {
		t.Errorf("same as old: %d, want 400", code)
	}
	if code, msg := change("old-pass", "new-pass-1"); code != http.StatusOK {
		t.Fatalf("change: %d %q", code, msg)
	}
	if _, err := svc.Login(auth.LoginRequest{Username: "chu", Password: "old-pass"}); err == nil {
		t.Error("old password still works")
	}
	if _, err := svc.Login(auth.LoginRequest{Username: "chu", Password: "new-pass-1"}); err != nil {
		t.Errorf("new password rejected: %v", err)
	}
}

func TestPhoneSignInAndUpdateMe(t *testing.T) {
	db := openTestDB(t, &auth.User{})
	svc := auth.NewService(auth.NewRepository(db), "test-secret", time.Hour)
	user, err := svc.CreateUser(auth.CreateUserRequest{Username: "admin", Password: "secret-123", Role: auth.RoleOwner})
	if err != nil {
		t.Fatal(err)
	}
	h := NewAuthMobileHandler(svc)
	call := func(handler http.HandlerFunc, method string, body any) (int, map[string]any) {
		b, _ := json.Marshal(body)
		req := asUser(httptest.NewRequest(method, "/", bytes.NewReader(b)), user.ID.String(), auth.RoleOwner)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler(w, req)
		var out map[string]any
		_ = json.NewDecoder(w.Body).Decode(&out)
		return w.Code, out
	}
	login := func(id string) int {
		code, _ := call(h.Login, http.MethodPost, map[string]string{"email": id, "password": "secret-123", "device_name": "t"})
		return code
	}

	if code := login("0912345678"); code != http.StatusUnauthorized {
		t.Fatalf("phone login before a phone is set: %d", code)
	}
	code, body := call(h.UpdateMe, http.MethodPut, map[string]string{"phone": "+84 912 345 678"})
	if code != http.StatusOK {
		t.Fatalf("update me: %d %v", code, body)
	}
	if phone := body["data"].(map[string]any)["phone"]; phone != "0912345678" {
		t.Errorf("phone in response = %v", phone)
	}
	if code := login("0912 345 678"); code != http.StatusOK {
		t.Errorf("phone login: %d", code)
	}
	if code := login("admin"); code != http.StatusOK {
		t.Errorf("username login still works: %d", code)
	}
	if code, body = call(h.UpdateMe, http.MethodPut, map[string]string{"phone": "abc"}); code != http.StatusBadRequest {
		t.Errorf("invalid phone: %d %v", code, body)
	}
	if code, _ = call(h.UpdateMe, http.MethodPut, map[string]string{"name": "  "}); code != http.StatusBadRequest {
		t.Errorf("empty name: %d", code)
	}
}

func TestCreateOrderIsIdempotentWithClientID(t *testing.T) {
	db := openTestDB(t, &customer.Customer{}, &product.Category{}, &product.Unit{}, &product.Product{},
		&product.ProductCategory{}, &order.Order{}, &order.OrderItem{}, &order.Payment{})
	p := product.Product{Name: "Rau", SellPrice: 10000, Active: true}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	orderSvc := order.NewService(order.NewRepository(db), product.NewRepository(db), nil)
	h := NewOrderMobileHandler(orderSvc, customer.NewService(customer.NewRepository(db)), time.UTC)
	post := func(body map[string]any) (int, map[string]any) {
		b, _ := json.Marshal(body)
		req := asUser(httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(b)), uuid.NewString(), auth.RoleStaff)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.Create(w, req)
		var out map[string]any
		_ = json.NewDecoder(w.Body).Decode(&out)
		return w.Code, out
	}
	clientID := uuid.NewString()
	soldAt := time.Now().Add(-26 * time.Hour).UTC().Truncate(time.Second)
	body := map[string]any{
		"client_id":  clientID,
		"created_at": soldAt.Format(time.RFC3339),
		"items":      []map[string]any{{"product_id": p.ID.String(), "quantity": 2}},
	}

	code, out := post(body)
	if code != http.StatusCreated {
		t.Fatalf("first send: %d %v", code, out)
	}
	if id := out["data"].(map[string]any)["id"]; id != clientID {
		t.Errorf("order id = %v, want the client id %s", id, clientID)
	}
	// The same sale sent again (lost response, offline sync retry).
	if code, out = post(body); code != http.StatusOK {
		t.Fatalf("retry: %d %v", code, out)
	}
	var count int64
	db.Model(&order.Order{}).Count(&count)
	if count != 1 {
		t.Fatalf("orders = %d, want 1 (no duplicate)", count)
	}
	var saved order.Order
	db.First(&saved, "id = ?", clientID)
	if !saved.CreatedAt.Equal(soldAt) {
		t.Errorf("created_at = %v, want the offline sale time %v", saved.CreatedAt, soldAt)
	}

	for name, at := range map[string]time.Time{
		"too old":   time.Now().Add(-40 * 24 * time.Hour),
		"in future": time.Now().Add(2 * time.Hour),
	} {
		body["client_id"] = uuid.NewString()
		body["created_at"] = at.Format(time.RFC3339)
		if code, out = post(body); code != http.StatusBadRequest {
			t.Errorf("%s: %d %v", name, code, out)
		}
	}
	body["client_id"] = "not-a-uuid"
	delete(body, "created_at")
	if code, _ = post(body); code != http.StatusBadRequest {
		t.Errorf("bad client id: %d", code)
	}
	// Without client_id it still works as before.
	delete(body, "client_id")
	if code, _ = post(body); code != http.StatusCreated {
		t.Errorf("plain create: %d", code)
	}
}
