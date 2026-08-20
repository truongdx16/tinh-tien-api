package app

import (
	"tinh-tien-api/internal/domain/auth"
	"tinh-tien-api/internal/domain/customer"
	"tinh-tien-api/internal/domain/product"
	"tinh-tien-api/internal/domain/settings"
	"tinh-tien-api/internal/pkg/config"
	"gorm.io/gorm"
)

type SeedOptions struct {
	OwnerUsername string
	OwnerPassword string
	OwnerFullName string
}

func DefaultSeedOptions() SeedOptions {
	return SeedOptions{
		OwnerUsername: "owner",
		OwnerPassword: "owner123",
		OwnerFullName: "Chủ hộ",
	}
}

func Seed(db *gorm.DB, cfg *config.Config, opts SeedOptions) error {
	if opts.OwnerUsername == "" {
		opts = DefaultSeedOptions()
	}

	authSvc := auth.NewService(auth.NewRepository(db), cfg.Auth.JWTSecret, cfg.Auth.TokenTTL)
	if err := authSvc.EnsureDefaultOwner(opts.OwnerUsername, opts.OwnerPassword, opts.OwnerFullName); err != nil {
		return err
	}

	settingsSvc := settings.NewService(settings.NewRepository(db))
	if err := settingsSvc.SeedDefaults(cfg.Shop.Name, cfg.Shop.Phone, cfg.Shop.Currency); err != nil {
		return err
	}

	// Seed walk-in customer CUS-0001 (Flutter convention)
	customerSvc := customer.NewService(customer.NewRepository(db))
	if err := customerSvc.EnsureWalkIn(); err != nil {
		return err
	}

	// Seed default units (kg, bó, túi)
	productSvc := product.NewService(product.NewRepository(db))
	defaultUnits := []struct{ name, slug string }{
		{"kg", "kg"},
		{"bó", "bo"},
		{"túi", "tui"},
		{"cái", "cai"},
		{"lít", "lit"},
	}
	for _, u := range defaultUnits {
		var existing product.Unit
		if err := db.Where("slug = ?", u.slug).Limit(1).Find(&existing).Error; err != nil {
			return err
		}
		if existing.Name == "" {
			if _, err := productSvc.CreateUnit(u.name, u.slug, 1); err != nil {
				return err
			}
		}
	}
	return nil
}
