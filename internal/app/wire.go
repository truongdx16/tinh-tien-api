package app

import (
	"encoding/base64"
	"log"
	"strings"

	"gorm.io/gorm"
	"tinh-tien-api/internal/domain/auth"
	"tinh-tien-api/internal/domain/crop"
	"tinh-tien-api/internal/domain/customer"
	"tinh-tien-api/internal/domain/expense"
	"tinh-tien-api/internal/domain/feedback"
	"tinh-tien-api/internal/domain/inventory"
	"tinh-tien-api/internal/domain/media"
	"tinh-tien-api/internal/domain/notification"
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

	notifySvc := notification.NewService(notification.NewRepository(db), BuildPushSender(cfg))

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
			Notify:   mobile.NewNotificationMobileHandler(notifySvc),
			App:      mobile.NewAppVersionMobileHandler(cfg.Media.UploadDir, cfg.Media.PublicBaseURL),
		},
	}
}

// BuildPushSender returns the FCM sender when Firebase credentials are
// configured; otherwise a no-op sender (reminders stored, not pushed).
func BuildPushSender(cfg *config.Config) notification.Sender {
	var (
		sa  *notification.ServiceAccount
		err error
	)
	switch {
	case cfg.Notify.CredentialsFile != "":
		sa, err = notification.LoadServiceAccountFile(cfg.Notify.CredentialsFile)
	case cfg.Notify.CredentialsBase64 != "":
		var raw []byte
		raw, err = base64.StdEncoding.DecodeString(strings.TrimSpace(cfg.Notify.CredentialsBase64))
		if err == nil {
			sa, err = notification.ParseServiceAccount(raw)
		}
	default:
		log.Printf("push notifications disabled: set FCM_CREDENTIALS_FILE or FCM_CREDENTIALS_BASE64")
		return notification.NoopSender{}
	}
	if err != nil {
		log.Printf("push notifications disabled: %v", err)
		return notification.NoopSender{}
	}
	log.Printf("push notifications enabled (Firebase project %s)", sa.ProjectID)
	return notification.NewFCMSender(sa)
}

// BuildReminderScheduler returns the daily planting reminder job, or nil
// when disabled in the config.
func BuildReminderScheduler(db *gorm.DB, cfg *config.Config) *notification.Scheduler {
	if cfg.Notify.Disabled {
		return nil
	}
	return &notification.Scheduler{
		Service:   notification.NewService(notification.NewRepository(db), BuildPushSender(cfg)),
		Schedules: planting.NewRepository(db),
		Location:  cfg.Location(),
		Hour:      cfg.Notify.ReminderHour,
		Interval:  cfg.Notify.CheckInterval,
	}
}
