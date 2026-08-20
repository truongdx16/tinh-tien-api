package app

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"tinh-tien-api/internal/domain/auth"
	"tinh-tien-api/internal/domain/crop"
	"tinh-tien-api/internal/domain/customer"
	"tinh-tien-api/internal/domain/expense"
	"tinh-tien-api/internal/domain/inventory"
	"tinh-tien-api/internal/domain/order"
	"tinh-tien-api/internal/domain/product"
	"tinh-tien-api/internal/domain/report"
	"tinh-tien-api/internal/domain/settings"
	"tinh-tien-api/internal/mobile"
	"tinh-tien-api/internal/pkg/httputil"
)

type Handlers struct {
	Auth     *auth.Handler
	Product  *product.Handler
	Customer *customer.Handler
	Inventory *inventory.Handler
	Order    *order.Handler
	Crop     *crop.Handler
	Expense  *expense.Handler
	Report   *report.Handler
	Settings *settings.Handler
	AuthSvc  *auth.Service
	Mobile   MobileHandlers
}

type MobileHandlers struct {
	Auth     *mobile.AuthMobileHandler
	Catalog  *mobile.CatalogMobileHandler
	Customer *mobile.CustomerMobileHandler
	Order    *mobile.OrderMobileHandler
	Planting *mobile.PlantingMobileHandler
	Media    *mobile.MediaMobileHandler
	Stats    *mobile.StatsMobileHandler
	Feedback *mobile.FeedbackMobileHandler
}

func NewRouter(h Handlers) http.Handler {
	r := chi.NewRouter()
	r.NotFound(NotFoundJSON)
	r.MethodNotAllowed(MethodNotAllowedJSON)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(RecoverJSON)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		httputil.OK(w, "service healthy", map[string]string{"status": "ok"})
	})

	specPath := os.Getenv("OPENAPI_PATH")
	if specPath == "" {
		specPath = "api/openapi.yaml"
	}
	r.Get("/openapi.yaml", OpenAPIHandler(specPath))
	r.Get("/docs", DocsHandler())

	// Serve uploaded files
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	// ---- /api/v1  Flutter mobile contract ----
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", h.Mobile.Auth.Login)

		r.Group(func(r chi.Router) {
			r.Use(auth.AuthMiddleware(h.AuthSvc))

			r.Post("/auth/logout", h.Mobile.Auth.Logout)
			r.Get("/auth/me", h.Mobile.Auth.Me)
			r.Get("/users", h.Mobile.Auth.ListUsers)

			r.Route("/categories", func(r chi.Router) {
				r.Get("/", h.Mobile.Catalog.ListCategories)
				r.Post("/", h.Mobile.Catalog.CreateCategory)
				r.Put("/{id}", h.Mobile.Catalog.UpdateCategory)
			})

			r.Route("/units", func(r chi.Router) {
				r.Get("/", h.Mobile.Catalog.ListUnits)
				r.Post("/", h.Mobile.Catalog.CreateUnit)
				r.Put("/{id}", h.Mobile.Catalog.UpdateUnit)
			})

			r.Route("/products", func(r chi.Router) {
				r.Get("/", h.Mobile.Catalog.ListProducts)
				r.Post("/", h.Mobile.Catalog.CreateProduct)
				r.Get("/{id}", h.Mobile.Catalog.GetProduct)
				r.Put("/{id}", h.Mobile.Catalog.UpdateProduct)
				r.Delete("/{id}", h.Mobile.Catalog.DeleteProduct)
			})

			r.Route("/customers", func(r chi.Router) {
				r.Get("/", h.Mobile.Customer.List)
				r.Post("/", h.Mobile.Customer.Create)
				r.Put("/{id}", h.Mobile.Customer.Update)
			})

			r.Route("/orders", func(r chi.Router) {
				r.Get("/", h.Mobile.Order.List)
				r.Post("/", h.Mobile.Order.Create)
				r.Get("/{id}", h.Mobile.Order.Get)
				r.Patch("/{id}/cancel", h.Mobile.Order.Cancel)
			})

			r.Route("/planting-schedules", func(r chi.Router) {
				r.Get("/", h.Mobile.Planting.List)
				r.Post("/", h.Mobile.Planting.Create)
				r.Get("/{id}", h.Mobile.Planting.Get)
				r.Put("/{id}", h.Mobile.Planting.Update)
				r.Delete("/{id}", h.Mobile.Planting.Delete)
			})

			r.Route("/media", func(r chi.Router) {
				r.Get("/", h.Mobile.Media.List)
				r.Post("/upload", h.Mobile.Media.Upload)
			})

			r.Route("/stats", func(r chi.Router) {
				r.Get("/revenue", h.Mobile.Stats.Revenue)
				r.Get("/products", h.Mobile.Stats.TopProducts)
				r.Get("/customers", h.Mobile.Stats.Customers)
			})

			r.Route("/feedback", func(r chi.Router) {
				r.Get("/", h.Mobile.Feedback.List)
				r.Post("/", h.Mobile.Feedback.Create)
			})
		})
	})

	// ---- /v1  existing admin/web routes ----
	r.Route("/v1", func(r chi.Router) {
		r.Post("/auth/login", h.Auth.Login)

		r.Group(func(r chi.Router) {
			r.Use(auth.AuthMiddleware(h.AuthSvc))

			r.Get("/settings", h.Settings.Get)
			r.Put("/settings", h.Settings.Update)

			r.Route("/users", func(r chi.Router) {
				r.With(auth.RequireOwner).Get("/", h.Auth.ListUsers)
				r.With(auth.RequireOwner).Post("/", h.Auth.CreateUser)
				r.With(auth.RequireOwner).Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
					h.Auth.GetUser(w, r, chi.URLParam(r, "id"))
				})
				r.With(auth.RequireOwner).Patch("/{id}", func(w http.ResponseWriter, r *http.Request) {
					h.Auth.UpdateUser(w, r, chi.URLParam(r, "id"))
				})
			})

			r.Route("/categories", func(r chi.Router) {
				r.Get("/", h.Product.ListCategories)
				r.Post("/", h.Product.CreateCategory)
				r.Get("/{id}", h.Product.GetCategory)
				r.Put("/{id}", h.Product.UpdateCategory)
				r.Delete("/{id}", h.Product.DeleteCategory)
			})

			r.Route("/products", func(r chi.Router) {
				r.Get("/", h.Product.ListProducts)
				r.Post("/", h.Product.CreateProduct)
				r.Get("/{id}", h.Product.GetProduct)
				r.Put("/{id}", h.Product.UpdateProduct)
				r.Delete("/{id}", h.Product.DeleteProduct)
			})

			r.Route("/customers", func(r chi.Router) {
				r.Get("/", h.Customer.List)
				r.Post("/", h.Customer.Create)
				r.Get("/{id}", h.Customer.Get)
				r.Put("/{id}", h.Customer.Update)
				r.Delete("/{id}", h.Customer.Delete)
			})

			r.Route("/inventory", func(r chi.Router) {
				r.Get("/", h.Inventory.ListBalances)
				r.Get("/balance", h.Inventory.GetBalance)
				r.Get("/movements", h.Inventory.ListMovements)
				r.Post("/adjustments", h.Inventory.Adjust)
			})

			r.Route("/orders", func(r chi.Router) {
				r.Get("/", h.Order.List)
				r.Post("/", h.Order.Create)
				r.Get("/{id}", h.Order.Get)
				r.Patch("/{id}/status", h.Order.UpdateStatus)
				r.Post("/{id}/payments", h.Order.AddPayment)
			})

			r.Route("/plots", func(r chi.Router) {
				r.Get("/", h.Crop.ListPlots)
				r.Post("/", h.Crop.CreatePlot)
				r.Get("/{id}", h.Crop.GetPlot)
				r.Put("/{id}", h.Crop.UpdatePlot)
				r.Delete("/{id}", h.Crop.DeletePlot)
			})

			r.Route("/crop-batches", func(r chi.Router) {
				r.Get("/", h.Crop.ListBatches)
				r.Post("/", h.Crop.CreateBatch)
				r.Get("/due-harvests", h.Crop.DueHarvests)
				r.Get("/{id}", h.Crop.GetBatch)
				r.Put("/{id}", h.Crop.UpdateBatch)
				r.Post("/{id}/activities", h.Crop.AddActivity)
				r.Post("/{id}/harvests", h.Crop.RecordHarvest)
			})

			r.Route("/expenses", func(r chi.Router) {
				r.Get("/", h.Expense.List)
				r.Post("/", h.Expense.Create)
				r.Get("/{id}", h.Expense.Get)
				r.Put("/{id}", h.Expense.Update)
				r.Delete("/{id}", h.Expense.Delete)
			})

			r.Route("/reports", func(r chi.Router) {
				r.Get("/sales", h.Report.Sales)
				r.Get("/products", h.Report.Products)
				r.Get("/crops", h.Report.Crops)
				r.Get("/receivables", h.Report.Receivables)
				r.Get("/low-stock", h.Report.LowStock)
				r.Get("/profit", h.Report.Profit)
			})
		})
	})

	return r
}
