package auth_test

import (
	"testing"
	"time"

	"tinh-tien-api/internal/domain/auth"
)

func TestServiceLoginAndToken(t *testing.T) {
	db := setupTestDB(t)
	repo := auth.NewRepository(db)
	svc := auth.NewService(repo, "test-secret", time.Hour)

	if err := svc.EnsureDefaultOwner("testowner", "password123", "Test Owner"); err != nil {
		t.Fatalf("ensure owner: %v", err)
	}

	resp, err := svc.Login(auth.LoginRequest{Username: "testowner", Password: "password123"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("expected token")
	}

	claims, err := svc.ParseToken(resp.Token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if claims.Username != "testowner" {
		t.Fatalf("expected testowner, got %s", claims.Username)
	}
}

func TestServiceInvalidLogin(t *testing.T) {
	db := setupTestDB(t)
	repo := auth.NewRepository(db)
	svc := auth.NewService(repo, "test-secret", time.Hour)
	_ = svc.EnsureDefaultOwner("testowner", "password123", "Test Owner")

	_, err := svc.Login(auth.LoginRequest{Username: "testowner", Password: "wrong"})
	if err == nil {
		t.Fatal("expected login error")
	}
}
