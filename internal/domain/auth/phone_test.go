package auth_test

import (
	"errors"
	"testing"
	"time"

	"tinh-tien-api/internal/domain/auth"
)

func TestNormalizePhone(t *testing.T) {
	same := []string{"0912345678", "0912 345 678", "091.234.5678", "+84912345678", "84912345678", "84-912-345-678", " (+84) 912 345 678 "}
	for _, in := range same {
		got, ok := auth.NormalizePhone(in)
		if !ok || got != "0912345678" {
			t.Errorf("NormalizePhone(%q) = %q, %v; want 0912345678", in, got, ok)
		}
	}
	for _, in := range []string{"", "admin", "jbonebit99@gmail.com", "12345", "0012345678", "091234567890"} {
		if got, ok := auth.NormalizePhone(in); ok {
			t.Errorf("NormalizePhone(%q) = %q, want not a phone", in, got)
		}
	}
}

func TestLoginWithPhone(t *testing.T) {
	svc := auth.NewService(auth.NewRepository(setupTestDB(t)), "test-secret", time.Hour)
	// The owner gets a phone through seed (OWNER_PHONE).
	if err := svc.EnsureDefaultOwner("admin", "password123", "Chủ hộ", "+84 912 345 678"); err != nil {
		t.Fatal(err)
	}

	for _, login := range []string{"admin", "0912345678", "0912 345 678", "+84912345678"} {
		if _, err := svc.Login(auth.LoginRequest{Username: login, Password: "password123"}); err != nil {
			t.Errorf("login with %q: %v", login, err)
		}
	}
	if _, err := svc.Login(auth.LoginRequest{Username: "0912345678", Password: "wrong"}); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("wrong password with phone: %v", err)
	}
	if _, err := svc.Login(auth.LoginRequest{Username: "0999999999", Password: "password123"}); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("unknown phone: %v", err)
	}
}

func TestUpdateProfilePhone(t *testing.T) {
	svc := auth.NewService(auth.NewRepository(setupTestDB(t)), "test-secret", time.Hour)
	_ = svc.EnsureDefaultOwner("admin", "password123", "Chủ hộ", "")
	staff, err := svc.CreateUser(auth.CreateUserRequest{Username: "nv1", Password: "password123", Phone: strPtr("0987654321")})
	if err != nil {
		t.Fatal(err)
	}
	owner, _ := svc.Login(auth.LoginRequest{Username: "admin", Password: "password123"})
	ownerID := owner.User.ID

	if _, err := svc.UpdateProfile(ownerID, nil, strPtr("12345")); !errors.Is(err, auth.ErrInvalidPhone) {
		t.Errorf("invalid phone: %v", err)
	}
	if _, err := svc.UpdateProfile(ownerID, nil, strPtr("+84 987 654 321")); !errors.Is(err, auth.ErrPhoneTaken) {
		t.Errorf("phone of another user: %v", err)
	}
	u, err := svc.UpdateProfile(ownerID, strPtr("  Trường  "), strPtr("0912 345 678"))
	if err != nil || u.Phone == nil || *u.Phone != "0912345678" || u.FullName != "Trường" {
		t.Fatalf("update profile: %+v, %v", u, err)
	}
	// Saving the same phone again is fine; "" removes it.
	if _, err := svc.UpdateProfile(ownerID, nil, strPtr("0912345678")); err != nil {
		t.Errorf("same phone again: %v", err)
	}
	if u, _ = svc.UpdateProfile(ownerID, nil, strPtr("")); u.Phone != nil {
		t.Errorf("phone not removed: %v", *u.Phone)
	}
	// Several users without a phone may coexist (unique index allows NULLs).
	if _, err := svc.UpdateProfile(staff.ID.String(), nil, strPtr("")); err != nil {
		t.Errorf("second user without phone: %v", err)
	}

	// Seeding an existing owner adds a phone only when it has none.
	if err := svc.EnsureDefaultOwner("admin", "ignored", "x", "0911111111"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Login(auth.LoginRequest{Username: "0911111111", Password: "password123"}); err != nil {
		t.Errorf("owner phone from seed: %v", err)
	}
}

func strPtr(s string) *string { return &s }
