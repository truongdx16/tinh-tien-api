package config

import (
	"strings"
	"testing"
)

func TestValidateProduction(t *testing.T) {
	good := Config{
		Auth:     AuthConfig{JWTSecret: strings.Repeat("a1", 16)},
		Database: DatabaseConfig{DSN: "u:p@tcp(127.0.0.1:3306)/db"},
		Media:    MediaConfig{PublicBaseURL: "https://7b.truongdx.dev"},
	}
	if err := good.validateProduction("production"); err != nil {
		t.Fatalf("good config rejected: %v", err)
	}

	bad := good
	bad.Auth.JWTSecret = "change-me-in-production-but-long-enough-xx"
	bad.Media.PublicBaseURL = "http://7b.truongdx.dev"
	err := bad.validateProduction("production")
	if err == nil || !strings.Contains(err.Error(), "JWT_SECRET") || !strings.Contains(err.Error(), "PUBLIC_BASE_URL") {
		t.Fatalf("expected JWT_SECRET and PUBLIC_BASE_URL problems, got %v", err)
	}

	// Outside production nothing is enforced (local dev).
	if err := bad.validateProduction(""); err != nil {
		t.Fatalf("dev config rejected: %v", err)
	}
}
