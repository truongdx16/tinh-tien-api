package app

import (
	"gorm.io/gorm"
	"tinh-tien-api/internal/domain/auth"
	"tinh-tien-api/internal/domain/crop"
	"tinh-tien-api/internal/domain/customer"
	"tinh-tien-api/internal/domain/expense"
	"tinh-tien-api/internal/domain/feedback"
	"tinh-tien-api/internal/domain/inventory"
	"tinh-tien-api/internal/domain/media"
	"tinh-tien-api/internal/domain/order"
	"tinh-tien-api/internal/domain/planting"
	"tinh-tien-api/internal/domain/product"
	"tinh-tien-api/internal/domain/report"
	"tinh-tien-api/internal/domain/settings"
	"tinh-tien-api/internal/mobile"
	"tinh-tien-api/internal/pkg/config"
)

func BuildHandlers(db *gorm.DB, cfg *config.Config) Handlers {
	authRepo := auth.NewRepository(db)
	authSvc := auth.NewService(authRepo, cfg.Auth.JWTSecret, cfg.Auth.TokenTTL)
	authHandler := auth.NewHandler(authSvc)

	productRepo := product.NewRepository(db)
	productSvc := product.NewService(productRepo)
	productHandler := product.NewHandler(productSvc)

	customerRepo := customer.NewRepository(db)
	customerSvc := customer.NewService(customerRepo)
	customerHandler := customer.NewHandler(customerSvc)

	invRepo := inventory.NewRepository(db)
	invSvc := inventory.NewService(invRepo)
	invHandler := inventory.NewHandler(invSvc)

	orderRepo := order.NewRepository(db)
	orderSvc := order.NewService(orderRepo, productRepo, invSvc)
	orderHandler := order.NewHandler(orderSvc)

	cropRepo := crop.NewRepository(db)
	cropSvc := crop.NewService(cropRepo, invSvc)
	cropHandler := crop.NewHandler(cropSvc)

	expenseRepo := expense.NewRepository(db)
	expenseSvc := expense.NewService(expenseRepo)
	expenseHandler := expense.NewHandler(expenseSvc)

	settingsRepo := settings.NewRepository(db)
	settingsSvc := settings.NewService(settingsRepo)
	settingsHandler := settings.NewHandler(settingsSvc)

	reportRepo := report.NewRepository(db)
	reportSvc := report.NewService(reportRepo, expenseSvc, invSvc, orderSvc)
	reportHandler := report.NewHandler(reportSvc)

	plantingRepo := planting.NewRepository(db)
	plantingSvc := planting.NewService(plantingRepo)

	mediaRepo := media.NewRepository(db)
	mediaSvc := media.NewService(mediaRepo)

	feedbackRepo := feedback.NewRepository(db)
	feedbackSvc := feedback.NewService(feedbackRepo)

	loc := cfg.Location()

	return Handlers{
		Auth:      authHandler,
		Product:   productHandler,
		Customer:  customerHandler,
		Inventory: invHandler,
		Order:     orderHandler,
		Crop:      cropHandler,
		Expense:   expenseHandler,
		Report:    reportHandler,
		Settings:  settingsHandler,
		AuthSvc:   authSvc,
		UploadDir: cfg.Media.UploadDir,
		Mobile: MobileHandlers{
			Auth:     mobile.NewAuthMobileHandler(authSvc),
			Catalog:  mobile.NewCatalogMobileHandler(productSvc),
			Customer: mobile.NewCustomerMobileHandler(customerSvc),
			Order:    mobile.NewOrderMobileHandler(orderSvc, customerSvc, loc),
			Planting: mobile.NewPlantingMobileHandler(plantingSvc, loc),
			Media:    mobile.NewMediaMobileHandler(mediaSvc, cfg.Media.PublicBaseURL, cfg.Media.UploadDir),
			Stats:    mobile.NewStatsMobileHandler(reportSvc, customerSvc, loc),
			Feedback: mobile.NewFeedbackMobileHandler(feedbackSvc),
		},
	}
}
