package mobile

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"tinh-tien-api/internal/domain/product"
	"tinh-tien-api/internal/mobile/adapter"
	"tinh-tien-api/internal/pkg/httputil"
)

// ---- DTOs for Flutter ----

type MobileCategoryDto struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status int    `json:"status"`
}

type MobileUnitDto struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Slug   *string `json:"slug"`
	Status int     `json:"status"`
}

type MobileProductDto struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	ImageURL      *string             `json:"image_url"`
	Price         string              `json:"price"`
	Quantity      string              `json:"quantity"`
	SalesQuantity string              `json:"sales_quantity"`
	Revenue       string              `json:"revenue"`
	Sensitivity   string              `json:"sensitivity"`
	Status        int                 `json:"status"`
	Unit          *MobileUnitDto      `json:"unit"`
	Categories    []MobileCategoryDto `json:"categories"`
}

type MobileProductWriteBody struct {
	Name        string   `json:"name"`
	ImageURL    *string  `json:"image_url"`
	Price       float64  `json:"price"`
	Quantity    float64  `json:"quantity"`
	Sensitivity *float64 `json:"sensitivity"`
	UnitID      *string  `json:"unit_id"`
	Status      int      `json:"status"`
	CategoryIDs []string `json:"category_ids"`
}

// ---- CatalogMobileHandler ----

type CatalogMobileHandler struct {
	svc *product.Service
}

func NewCatalogMobileHandler(svc *product.Service) *CatalogMobileHandler {
	return &CatalogMobileHandler{svc: svc}
}

// ---------- Categories ----------

func (h *CatalogMobileHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	items, _, err := h.svc.ListCategories(1, 1000)
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "failed to list categories")
		return
	}
	dtos := make([]MobileCategoryDto, 0, len(items))
	for _, c := range items {
		dtos = append(dtos, toCategoryDto(c))
	}
	adapter.OK(w, "categories retrieved", dtos)
}

func (h *CatalogMobileHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string `json:"name"`
		Status int    `json:"status"`
	}
	if err := httputil.Decode(r, &body); err != nil {
		adapter.Fail(w, http.StatusBadRequest, "invalid request body")
		return
	}
	item, err := h.svc.CreateCategory(product.CreateCategoryRequest{Name: body.Name})
	if err != nil {
		adapter.Fail(w, http.StatusBadRequest, "failed to create category")
		return
	}
	adapter.Created(w, "category created", toCategoryDto(*item))
}

func (h *CatalogMobileHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Name   *string `json:"name"`
		Status *int    `json:"status"`
	}
	if err := httputil.Decode(r, &body); err != nil {
		adapter.Fail(w, http.StatusBadRequest, "invalid request body")
		return
	}
	item, err := h.svc.UpdateCategory(id, product.UpdateCategoryRequest{Name: body.Name})
	if err != nil {
		if errors.Is(err, product.ErrNotFound) {
			adapter.Fail(w, http.StatusNotFound, "category not found")
			return
		}
		adapter.Fail(w, http.StatusInternalServerError, "failed to update category")
		return
	}
	adapter.OK(w, "category updated", toCategoryDto(*item))
}

// ---------- Units ----------

func (h *CatalogMobileHandler) ListUnits(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListUnits()
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "failed to list units")
		return
	}
	dtos := make([]MobileUnitDto, 0, len(items))
	for _, u := range items {
		dtos = append(dtos, toUnitDto(u))
	}
	adapter.OK(w, "units retrieved", dtos)
}

func (h *CatalogMobileHandler) CreateUnit(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string  `json:"name"`
		Slug   *string `json:"slug"`
		Status int     `json:"status"`
	}
	if err := httputil.Decode(r, &body); err != nil {
		adapter.Fail(w, http.StatusBadRequest, "invalid request body")
		return
	}
	slug := ""
	if body.Slug != nil {
		slug = *body.Slug
	}
	item, err := h.svc.CreateUnit(body.Name, slug, body.Status)
	if err != nil {
		adapter.Fail(w, http.StatusBadRequest, "failed to create unit")
		return
	}
	adapter.Created(w, "unit created", toUnitDto(*item))
}

func (h *CatalogMobileHandler) UpdateUnit(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Name   *string `json:"name"`
		Slug   *string `json:"slug"`
		Status *int    `json:"status"`
	}
	if err := httputil.Decode(r, &body); err != nil {
		adapter.Fail(w, http.StatusBadRequest, "invalid request body")
		return
	}
	item, err := h.svc.UpdateUnit(id, body.Name, body.Slug, body.Status)
	if err != nil {
		if errors.Is(err, product.ErrNotFound) {
			adapter.Fail(w, http.StatusNotFound, "unit not found")
			return
		}
		adapter.Fail(w, http.StatusInternalServerError, "failed to update unit")
		return
	}
	adapter.OK(w, "unit updated", toUnitDto(*item))
}

// ---------- Products ----------

func (h *CatalogMobileHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page := 1
	perPage := 20
	if p := q.Get("per_page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			perPage = v
		}
	}
	if p := q.Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	listQ := product.ProductListQuery{
		CategoryID: q.Get("category_id"),
		Search:     q.Get("search"),
		Page:       page,
		PageSize:   perPage,
	}
	if s := q.Get("status"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			active := v == 1
			listQ.Active = &active
		}
	}

	items, total, err := h.svc.ListProducts(listQ)
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "failed to list products")
		return
	}
	dtos := make([]MobileProductDto, 0, len(items))
	for _, p := range items {
		dtos = append(dtos, toProductDto(p))
	}
	adapter.OKPaginated(w, "products retrieved", dtos, page, total, perPage)
}

func (h *CatalogMobileHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	item, err := h.svc.GetProduct(id)
	if err != nil {
		if errors.Is(err, product.ErrNotFound) {
			adapter.Fail(w, http.StatusNotFound, "product not found")
			return
		}
		adapter.Fail(w, http.StatusInternalServerError, "failed to get product")
		return
	}
	adapter.OK(w, "product retrieved", toProductDto(*item))
}

func (h *CatalogMobileHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var body MobileProductWriteBody
	if err := httputil.Decode(r, &body); err != nil {
		adapter.Fail(w, http.StatusBadRequest, "invalid request body")
		return
	}
	sensitivity := "0"
	if body.Sensitivity != nil {
		sensitivity = strconv.FormatFloat(*body.Sensitivity, 'f', -1, 64)
	}
	imageURL := ""
	if body.ImageURL != nil {
		imageURL = *body.ImageURL
	}
	req := product.CreateProductRequest{
		UnitID:      body.UnitID,
		Name:        body.Name,
		ImageURL:    imageURL,
		Sensitivity: sensitivity,
		SellPrice:   int64(body.Price),
		Active:      body.Status == 1,
		CategoryIDs: body.CategoryIDs,
	}
	item, err := h.svc.CreateProduct(req)
	if err != nil {
		adapter.Fail(w, http.StatusBadRequest, "failed to create product")
		return
	}
	// reload with associations
	item, _ = h.svc.GetProduct(item.ID.String())
	adapter.Created(w, "product created", toProductDto(*item))
}

func (h *CatalogMobileHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body MobileProductWriteBody
	if err := httputil.Decode(r, &body); err != nil {
		adapter.Fail(w, http.StatusBadRequest, "invalid request body")
		return
	}
	sensitivity := "0"
	if body.Sensitivity != nil {
		sensitivity = strconv.FormatFloat(*body.Sensitivity, 'f', -1, 64)
	}
	imageURL := ""
	if body.ImageURL != nil {
		imageURL = *body.ImageURL
	}
	price := int64(body.Price)
	active := body.Status == 1
	req := product.UpdateProductRequest{
		UnitID:      body.UnitID,
		Name:        &body.Name,
		ImageURL:    &imageURL,
		Sensitivity: &sensitivity,
		SellPrice:   &price,
		Active:      &active,
		CategoryIDs: body.CategoryIDs,
	}
	item, err := h.svc.UpdateProduct(id, req)
	if err != nil {
		if errors.Is(err, product.ErrNotFound) {
			adapter.Fail(w, http.StatusNotFound, "product not found")
			return
		}
		adapter.Fail(w, http.StatusInternalServerError, "failed to update product")
		return
	}
	item, _ = h.svc.GetProduct(item.ID.String())
	adapter.OK(w, "product updated", toProductDto(*item))
}

func (h *CatalogMobileHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteProduct(id); err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "failed to delete product")
		return
	}
	adapter.OK(w, "product deleted", nil)
}

// ---------- Helpers ----------

func toCategoryDto(c product.Category) MobileCategoryDto {
	return MobileCategoryDto{
		ID:     c.ID.String(),
		Name:   c.Name,
		Status: adapter.BoolToStatus(c.Active),
	}
}

func toUnitDto(u product.Unit) MobileUnitDto {
	slug := u.Slug
	var slugPtr *string
	if slug != "" {
		slugPtr = &slug
	}
	return MobileUnitDto{
		ID:     u.ID.String(),
		Name:   u.Name,
		Slug:   slugPtr,
		Status: adapter.BoolToStatus(u.Active),
	}
}

func toProductDto(p product.Product) MobileProductDto {
	imageURL := p.ImageURL
	var imagePtr *string
	if imageURL != "" {
		imagePtr = &imageURL
	}
	sensitivity := p.Sensitivity
	if sensitivity == "" {
		sensitivity = "0"
	}

	cats := make([]MobileCategoryDto, 0, len(p.Categories))
	for _, c := range p.Categories {
		cats = append(cats, toCategoryDto(c))
	}
	// fallback to single category
	if len(cats) == 0 && p.Category != nil {
		cats = append(cats, toCategoryDto(*p.Category))
	}

	var unitDto *MobileUnitDto
	if p.UnitRef != nil {
		u := toUnitDto(*p.UnitRef)
		unitDto = &u
	}

	return MobileProductDto{
		ID:            p.ID.String(),
		Name:          p.Name,
		ImageURL:      imagePtr,
		Price:         adapter.MoneyStr(p.SellPrice),
		Quantity:      "0",
		SalesQuantity: "0",
		Revenue:       "0",
		Sensitivity:   sensitivity,
		Status:        adapter.BoolToStatus(p.Active),
		Unit:          unitDto,
		Categories:    cats,
	}
}
