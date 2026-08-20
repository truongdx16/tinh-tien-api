# Prompt: Align `tinh-tien-api` with Flutter Mobile Client Contract

> **Mục đích:** Copy toàn bộ nội dung trong section **PROMPT** bên dưới vào AI agent / task để chỉnh sửa backend Go API cho khớp với Flutter app.
>
> **Nguồn contract (Flutter):** `/Users/ominext/dev99/tinh_tien`
> **Target API (Go):** `/Users/ominext/dev99/tinh-tien-api`

---

## PROMPT (copy từ đây)

```markdown
# Task: Align tinh-tien-api with Flutter client contract (tinh_tien)

## 1. Bối cảnh

Mobile Flutter app `tinh_tien` đã implement Retrofit client và DTOs, expect một REST API contract cố định. Backend Go hiện tại (`tinh-tien-api`) có **domain model khác hoàn toàn** — không tương thích trực tiếp.

**Mục tiêu:** Flutter app chỉ cần đổi `API_BASE_URL` sang Go API mà **không sửa DTO / Retrofit interface**.

**Nguyên tắc:**
- Flutter DTO field names là **source of truth** — API phải adapt, không đổi tên field Flutter.
- Có thể giữ domain Go hiện tại (`/v1`, plots, crop-batches...) song song; thêm **compatibility layer** `/api/v1` map sang tables hiện có hoặc tables mới.
- Money/quantity trả về **decimal string** (ví dụ `"15000.5"`), không phải int.
- Status flags dùng **int** (`1` = active, `0` = inactive) trừ khi ghi chú khác.
- Dates trả về **ISO 8601 string** (`YYYY-MM-DD` hoặc full datetime).

---

## 2. Flutter client hiện expect gì?

### 2.1 Base URL & config

```
Default: http://127.0.0.1:8000/api/v1
Env:     API_BASE_URL (compile-time)
Auth:    Bearer token (Dio interceptor)
Timeout: connect 15s, receive 30s
```

File tham chiếu:
- `lib/core/config/api_config.dart`
- `lib/app/data/datasources/api_client.dart`
- `lib/app/data/datasources/api/interceptors/api_response_interceptor.dart`

### 2.2 Response envelope

Flutter interceptor unwrap **chỉ field `data`**:

```json
{
  "success": true,
  "message": "...",
  "data": <T>
}
```

Go API hiện có thêm `error`, `pagination` top-level — Flutter **bỏ qua** nếu không nằm trong `data`.

**Pagination** (products, customers): Flutter expect `data` chứa:

```json
{
  "data": [...],
  "current_page": 1
}
```

**List endpoints** trả array trực tiếp trong `data`: categories, units, orders, planting-schedules, media, feedback, stats products/customers.

### 2.3 Toàn bộ endpoints Flutter gọi

| Method | Path | Query / Body | Response `data` |
|--------|------|--------------|-----------------|
| POST | `/auth/login` | LoginRequestDto | LoginResponseDto |
| POST | `/auth/logout` | — | void |
| GET | `/auth/me` | — | UserDto |
| GET | `/users` | `status?` | `UserDto[]` |
| GET | `/products` | `status?`, `search?`, `category_id?`, `per_page?` | `PaginatedDto<ProductDto>` |
| GET | `/products/{id}` | — | ProductDto |
| POST | `/products` | Map (productToApi) | ProductDto |
| PUT | `/products/{id}` | Map | ProductDto |
| DELETE | `/products/{id}` | — | void |
| GET | `/categories` | — | `CategoryDto[]` |
| POST | `/categories` | Map | CategoryDto |
| PUT | `/categories/{id}` | Map | CategoryDto |
| GET | `/units` | — | `UnitDto[]` |
| POST | `/units` | Map | UnitDto |
| PUT | `/units/{id}` | Map | UnitDto |
| GET | `/customers` | `search?`, `per_page?` | `PaginatedDto<CustomerDto>` |
| POST | `/customers` | Map | CustomerDto |
| PUT | `/customers/{id}` | Map | CustomerDto |
| GET | `/orders` | `from`* , `to`* , `status?` | `OrderDto[]` |
| GET | `/orders/{id}` | — | OrderDto |
| POST | `/orders` | Map (orderToApi) | OrderDto |
| PATCH | `/orders/{id}/cancel` | — | OrderDto |
| GET | `/planting-schedules` | — | `PlantingScheduleDto[]` |
| GET | `/planting-schedules/{id}` | — | PlantingScheduleDto |
| POST | `/planting-schedules` | Map | PlantingScheduleDto |
| PUT | `/planting-schedules/{id}` | Map | PlantingScheduleDto |
| DELETE | `/planting-schedules/{id}` | — | void |
| GET | `/media` | — | `MediaDto[]` |
| POST | `/media/upload` | multipart: `file`, `entity_type?`, `entity_id?` | MediaDto |
| GET | `/stats/revenue` | `from`* , `to`* | RevenueStatsDto |
| GET | `/stats/products` | `limit?` | `TopProductStatsDto[]` |
| GET | `/stats/customers` | `from`* , `to`* | `CustomerReportDto[]` |
| GET | `/feedback` | — | `FeedbackDto[]` |
| POST | `/feedback` | FeedbackRequestDto | FeedbackDto |

\* required

**Endpoints Flutter define nhưng chưa dùng trong repository:** `GET /auth/me`, `GET/DELETE /products/{id}`, `GET /orders/{id}`, `PATCH /orders/{id}/cancel`, `GET /planting-schedules/{id}` — vẫn nên implement.

**Endpoints Flutter KHÔNG có:** category delete, unit delete, customer delete, order update (status), inventory, expenses, plots, crop-batches.

---

## 3. Toàn bộ DTO schemas (Flutter source of truth)

### Auth

**LoginRequestDto** (POST body):
```json
{
  "email": "string",
  "password": "string",
  "device_name": "mobile"
}
```

**LoginResponseDto**:
```json
{
  "token": "string",
  "user": { UserDto }
}
```

**UserDto**:
```json
{
  "id": "string (uuid)",
  "name": "string",
  "email": "string",
  "store_id": "string|null",
  "role": "string|null",
  "status": 1
}
```

### Catalog

**CategoryDto**:
```json
{ "id": "string", "name": "string", "status": 1 }
```

**UnitDto**:
```json
{ "id": "string", "name": "string", "slug": "string|null", "status": 1 }
```

**ProductDto**:
```json
{
  "id": "string",
  "name": "string",
  "image_url": "string|null",
  "price": "0",
  "quantity": "0",
  "sales_quantity": "0",
  "revenue": "0",
  "sensitivity": "0.1",
  "status": 1,
  "unit": { UnitDto } | null,
  "categories": [ CategoryDto ]
}
```

**Product write body** (`productToApi`):
```json
{
  "name": "string",
  "image_url": "string|null",
  "price": number,
  "quantity": number,
  "sensitivity": number,
  "unit_id": "string|null",
  "status": 1,
  "category_ids": ["string"]
}
```

**Category write body**:
```json
{ "name": "string", "status": 1 }
```

**Unit write body**:
```json
{ "name": "string", "slug": "string|null", "status": 1 }
```

### Customer

**CustomerDto**:
```json
{
  "id": "string",
  "code": "string|null",
  "name": "string",
  "phone": "string|null",
  "address": "string|null",
  "status": 1,
  "is_walk_in": false
}
```

**Customer write body**:
```json
{
  "code": "string|null",
  "name": "string",
  "phone": "string|null",
  "address": "string|null",
  "status": 1,
  "is_walk_in": false
}
```

**Walk-in guest convention:** Flutter dùng `id == "CUS-0001"` hoặc `code == "CUS-0001"` cho khách vãng lai. Cần seed customer này.

### Order

**OrderDto**:
```json
{
  "id": "string",
  "customer_id": "string|null",
  "discount": "0",
  "revenue": "0",
  "paid_amount": "0",
  "status": 1,
  "created_at": "string|null",
  "updated_at": "string|null",
  "customer": { CustomerDto } | null,
  "items": [
    {
      "product_id": "string|null",
      "product_name": "string",
      "unit_name": "string|null",
      "unit_price": "string",
      "quantity": "string",
      "line_total": "string"
    }
  ]
}
```

**Order write body** (`orderToApi`):
```json
{
  "id": "string|null",
  "customer_id": "string|null",
  "discount": number,
  "paid_amount": number,
  "items": [
    { "product_id": "string", "quantity": number }
  ]
}
```

Notes:
- Server tính `unit_price`, `line_total`, `revenue` từ product price × quantity − discount.
- `paid_amount` chỉ gửi khi thanh toán một phần (`paid_amount < revenue`).

**Order status int mapping** (Flutter UI dùng — cần document và implement nhất quán):

| Flutter int | Ý nghĩa UI | Gợi ý map Go status |
|-------------|------------|----------------------|
| 0 | Chờ xử lý (pending) | draft |
| 1 | Đang xử lý (processing) | confirmed / packed |
| 2 | Hoàn thành (completed) | delivered |
| 3 | Đã hủy (cancelled) | cancelled |

### Planting schedule (nested document model)

**PlantingScheduleDto**:
```json
{
  "id": "string",
  "vegetable_name": "string",
  "seed_type": "string|null",
  "planting_date": "YYYY-MM-DD|null",
  "expected_harvest_date": "YYYY-MM-DD|null",
  "actual_harvest_date": "YYYY-MM-DD|null",
  "area": "string|null",
  "seed_quantity": "string|null",
  "seed_unit": "string|null",
  "location": "string|null",
  "expected_yield": "string|null",
  "actual_yield": "string|null",
  "seed_cost": "string|null",
  "notes": "string|null",
  "status": 0,
  "is_harvested": false,
  "spray_tasks": [ SprayTaskDto ],
  "fertilize_tasks": [ FertilizeTaskDto ],
  "planting_tasks": [ PlantingTaskDto ]
}
```

**SprayTaskDto**:
```json
{
  "spray_date": "YYYY-MM-DD",
  "pesticide_name": "string|null",
  "pesticide_type": "string|null",
  "dosage": "string|null",
  "quarantine_date": "YYYY-MM-DD|null",
  "quarantine_days": "int|null",
  "cost": "string|null",
  "description": "string|null"
}
```

**FertilizeTaskDto**:
```json
{
  "fertilize_date": "YYYY-MM-DD",
  "fertilizer_name": "string|null",
  "fertilizer_type": "string|null",
  "amount": "string|null",
  "unit": "string|null",
  "application_method": "string|null",
  "application_number": "int|null",
  "cost": "string|null",
  "description": "string|null"
}
```

**PlantingTaskDto** (other tasks):
```json
{
  "task_name": "string",
  "task_date": "YYYY-MM-DD",
  "task_type": "string|null",
  "status": 0,
  "cost": "string|null",
  "description": "string|null"
}
```

**PlantingSchedule write body** (`plantingScheduleToApi`):
- Tất cả fields trên (trừ `id` khi create)
- Nested arrays: `spray_tasks`, `fertilize_tasks`, `planting_tasks`
- Dates gửi dạng `YYYY-MM-DD`

**Business logic quan trọng:** Flutter **chặn thu hoạch** nếu còn spray task chưa đủ `quarantine_date`. API phải persist đúng `quarantine_date` và `quarantine_days`.

**Optional nhưng mapper đọc:** `created_at`, `updated_at` trên PlantingScheduleDto (hiện DTO chưa declare nhưng domain mapper expect).

### Media

**MediaDto**:
```json
{
  "id": "string",
  "file_url": "string",
  "file_name": "string|null",
  "file_path": "string|null"
}
```

Upload: `POST /media/upload` multipart
- `file` (required)
- `entity_type` (optional, Flutter dùng `"product"`)
- `entity_id` (optional)

Mapper còn đọc (optional, chưa có trong DTO): `file_size`, `mime_type`, `md5_hash`.

### Stats

**RevenueStatsDto**:
```json
{
  "daily": [
    {
      "summary_date": "YYYY-MM-DD",
      "order_count": 0,
      "subtotal": "0",
      "discount": "0",
      "revenue": "0"
    }
  ],
  "totals": {
    "order_count": 0,
    "subtotal": "0",
    "discount": "0",
    "revenue": "0"
  }
}
```

**TopProductStatsDto**:
```json
{
  "id": "string",
  "name": "string",
  "sales_quantity": "0",
  "revenue": "0",
  "price": "0"
}
```

**CustomerReportDto**:
```json
{
  "customer_id": "string|null",
  "customer_name": "string",
  "order_count": 0,
  "revenue": "0",
  "debt": "0"
}
```

### Feedback

**FeedbackRequestDto** (POST):
```json
{
  "content": "string",
  "rating": "int|null",
  "full_name": "string|null"
}
```

**FeedbackDto**:
```json
{
  "id": 1,
  "content": "string",
  "rating": "int|null",
  "created_at": "string|null",
  "user_id": "int|null"
}
```

---

## 4. Go API hiện có (`tinh-tien-api`) — tóm tắt

**Stack:** Go 1.19, chi, GORM, MariaDB, JWT (username/password), port `:2170`

**Base path:** `/v1` (KHÔNG có `/api` prefix)

**Auth:**
- `POST /v1/auth/login` — `{ username, password }` → `{ token, expires_in, user: {id, username, full_name, role} }`
- Không có `/auth/logout`, `/auth/me`

**Entities chính:** users, customers, categories, products, balances, movements, orders, order_items, payments, plots, crop_batches, crop_activities, harvests, expenses, settings

**Money:** int64 VND (đồng), không phải decimal string

**Product:** single `category_id`, `unit` string, `sell_price`/`cost_price` int — không có image_url, sensitivity, sales_quantity, revenue aggregate

**Customer:** `name, phone, address, notes, active` — không có `code`, `is_walk_in`

**Order:** status string enum, fulfillment_type, delivery_address, payments — không có `discount` field riêng

**Planting:** plots + crop-batches + activities + harvests — **KHÔNG có** planting-schedules nested spray/fertilize model

**Reports:** `/v1/reports/sales|products|crops|receivables|low-stock|profit` — path/shape khác Flutter `/stats/*`

**Thiếu hoàn toàn:** units CRUD, media, feedback, planting-schedules

---

## 5. Gap analysis chi tiết

### 5.1 Infrastructure gaps

| # | Flutter expect | Go API hiện có | Action |
|---|----------------|----------------|--------|
| 1 | Base `/api/v1` | Base `/v1` | Mount route group `/api/v1` |
| 2 | Envelope `{success,message,data}` | Có + `error`, `pagination` | OK nếu Flutter chỉ đọc `data` |
| 3 | Pagination `current_page` | `page`, `page_size`, `total` | Adapter trả `current_page` |
| 4 | Money = decimal string | int64 VND | Format int → string khi response; parse string → int khi request |
| 5 | Status = int | active bool / status string | Map bool/string ↔ int |

### 5.2 Auth gaps

| # | Flutter | Go | Action |
|---|---------|-----|--------|
| 1 | Login bằng `email` | Login bằng `username` | Accept `email` as username OR add email column |
| 2 | `device_name` field | Không có | Ignore (no-op) |
| 3 | `POST /auth/logout` | Không có | Implement no-op hoặc token blacklist |
| 4 | `GET /auth/me` | Không có | Implement |
| 5 | UserDto: `name, email, store_id, status:int` | `full_name, username, active:bool` | Field mapping adapter |

### 5.3 Catalog gaps

| # | Flutter | Go | Action |
|---|---------|-----|--------|
| 1 | Units entity + CRUD | Product.unit string | **New `units` table** + `products.unit_id` FK |
| 2 | Product many categories | Single category_id | Pivot `product_categories` OR return `[one]` |
| 3 | Product: image_url, sensitivity | Không có | Add columns |
| 4 | Product: quantity | balances table | Join balances.quantity |
| 5 | Product: sales_quantity, revenue | Không có | Aggregate from order_items |
| 6 | Product: price string | sell_price int | Map sell_price → price string |
| 7 | Category: status int | description, soft-delete | Map active/deleted → status |

### 5.4 Customer gaps

| # | Flutter | Go | Action |
|---|---------|-----|--------|
| 1 | `code` | Không có | Add column |
| 2 | `is_walk_in` | Không có | Add column |
| 3 | `status: int` | `active: bool` | Map |
| 4 | Walk-in `CUS-0001` | Không seed | Seed customer |

### 5.5 Order gaps

| # | Flutter | Go | Action |
|---|---------|-----|--------|
| 1 | `discount` field | Không có (subtotal/total) | Add discount column or derive |
| 2 | `revenue` string | `total` int | Map total → revenue string |
| 3 | `status: int` | status string enum | Mapping table (section 3) |
| 4 | `PATCH .../cancel` | `PATCH .../status` | Alias endpoint |
| 5 | Create chỉ gửi product_id+qty | Create có unit_price optional | Server lấy price từ product |
| 6 | Filter `from/to` required | Có from/to | OK |
| 7 | Filter `status` int | status string query | Map int → string filter |

### 5.6 Planting gaps (LỚN NHẤT)

| # | Flutter | Go | Action |
|---|---------|-----|--------|
| 1 | `/planting-schedules` nested doc | plots + crop-batches | **New tables** hoặc JSON doc store |
| 2 | spray_tasks với quarantine | crop_activities generic | Không map được — cần schema riêng |
| 3 | fertilize_tasks chi tiết | Không có | New table |
| 4 | planting_tasks (other) | activities type=other | Partial map, thiếu fields |
| 5 | is_harvested bool | crop batch status | Map harvested status |
| 6 | area, seed_cost, yield... | Không có | New columns |

**Khuyến nghị:** Tạo tables riêng `planting_schedules`, `spray_tasks`, `fertilize_tasks`, `planting_tasks`. Giữ plots/crop-batches cho admin/web nếu cần.

### 5.7 Media gaps

| # | Flutter | Go | Action |
|---|---------|-----|--------|
| 1 | GET/POST /media | Không có | Implement file storage + DB |
| 2 | Link image → product.image_url | Không có | Update product on upload |

### 5.8 Stats gaps

| Flutter path | Go equivalent | Gap |
|--------------|---------------|-----|
| GET /stats/revenue | GET /reports/sales | Shape khác: cần daily[] + totals + discount |
| GET /stats/products | GET /reports/products | Field names khác |
| GET /stats/customers | GET /reports/receivables | Cần aggregate by customer + debt |

### 5.9 Feedback gaps

| # | Flutter | Go | Action |
|---|---------|-----|--------|
| 1 | GET/POST /feedback | Không có | New table + handlers |

---

## 6. Yêu cầu implementation

### Phase 1 — Unblock cơ bản (P0)

1. Mount `/api/v1` route group
2. Auth: login (email→username), logout, me
3. Envelope + pagination adapter
4. Money/status/date formatting helpers

### Phase 2 — Catalog (P0)

5. Units table + CRUD
6. Products: add columns, join unit/balance/aggregates, multi-category
7. Categories CRUD với status int

### Phase 3 — Sales (P0)

8. Customers: code, is_walk_in, status mapping, seed CUS-0001
9. Orders: discount, status int mapping, cancel endpoint, create from product_id+qty

### Phase 4 — Planting (P1)

10. Planting schedules full nested model
11. Quarantine logic persistence

### Phase 5 — Media & Reports & Feedback (P1)

12. Media upload + list
13. Stats endpoints (adapter từ reports)
14. Feedback CRUD

---

## 7. Implementation notes

### Route mounting

```go
r.Route("/api/v1", func(r chi.Router) {
    // mobile-compatible handlers
})
// Keep existing /v1 for admin/other clients
```

### Response helper example

```go
type Envelope struct {
    Success bool        `json:"success"`
    Message string      `json:"message"`
    Data    interface{} `json:"data"`
}

func OK(w http.ResponseWriter, data interface{}) {
    json.NewEncoder(w).Encode(Envelope{Success: true, Message: "OK", Data: data})
}
```

### Money formatting

```go
func moneyStr(v int64) string { return strconv.FormatInt(v, 10) }
func parseMoney(s string) (int64, error) { /* parse decimal string to int64 VND */ }
```

### Product response builder (pseudo)

```
price        = moneyStr(product.SellPrice)
quantity     = moneyStr(balance.Quantity)  // float → string
sales_qty    = aggregate SUM(order_items.quantity) WHERE product_id
revenue      = aggregate SUM(order_items.line_total)
unit         = UnitDto from units table
categories   = []CategoryDto from pivot
image_url    = product.ImageURL
sensitivity  = product.Sensitivity (string)
```

### Order create flow

1. Validate items (product_id, quantity)
2. Load product prices
3. Compute line_total, subtotal, apply discount
4. Create order + order_items
5. Deduct inventory (existing Go logic)
6. Return OrderDto with nested customer + items

### Planting schedule schema suggestion

```sql
planting_schedules (
  id UUID PK,
  vegetable_name VARCHAR NOT NULL,
  seed_type VARCHAR,
  planting_date DATE,
  expected_harvest_date DATE,
  actual_harvest_date DATE,
  area DECIMAL,
  seed_quantity DECIMAL,
  seed_unit VARCHAR,
  location VARCHAR,
  expected_yield DECIMAL,
  actual_yield DECIMAL,
  seed_cost DECIMAL,
  notes TEXT,
  status INT DEFAULT 0,
  is_harvested BOOL DEFAULT FALSE,
  created_at, updated_at, deleted_at
)

spray_tasks (
  id UUID PK,
  schedule_id UUID FK,
  spray_date DATE NOT NULL,
  pesticide_name VARCHAR,
  pesticide_type VARCHAR,
  dosage VARCHAR,
  quarantine_date DATE,
  quarantine_days INT,
  cost DECIMAL,
  description TEXT
)

-- Similar for fertilize_tasks, planting_tasks
```

On write: replace nested arrays (delete old + insert new) OR upsert by index.

---

## 8. Seed data required

```yaml
users:
  - username: owner
    email: owner@example.com   # or map username as email
    password: owner123
    role: owner
    status: 1

customers:
  - id: CUS-0001
    code: CUS-0001
    name: "Khách vãng lai"
    is_walk_in: true
    status: 1

units:
  - name: "kg", slug: "kg", status: 1
  - name: "bó", slug: "bo", status: 1
  - name: "túi", slug: "tui", status: 1
```

---

## 9. Integration test checklist (curl)

```bash
# 1. Login
curl -X POST http://localhost:2170/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"owner","password":"owner123","device_name":"mobile"}'

# 2. Me
curl http://localhost:2170/api/v1/auth/me -H "Authorization: Bearer $TOKEN"

# 3. List units
curl http://localhost:2170/api/v1/units -H "Authorization: Bearer $TOKEN"

# 4. List products (paginated)
curl "http://localhost:2170/api/v1/products?per_page=20" -H "Authorization: Bearer $TOKEN"

# 5. Create product
curl -X POST http://localhost:2170/api/v1/products \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"Rau muống","price":15000,"quantity":100,"sensitivity":0.1,"unit_id":"...","status":1,"category_ids":[]}'

# 6. Create order
curl -X POST http://localhost:2170/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"customer_id":"CUS-0001","discount":0,"items":[{"product_id":"...","quantity":2}]}'

# 7. List orders
curl "http://localhost:2170/api/v1/orders?from=2026-01-01&to=2026-12-31" \
  -H "Authorization: Bearer $TOKEN"

# 8. Create planting schedule with spray quarantine
curl -X POST http://localhost:2170/api/v1/planting-schedules \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"vegetable_name":"Cải xanh","planting_date":"2026-01-01","spray_tasks":[{"spray_date":"2026-01-15","quarantine_days":7}]}'

# 9. Stats revenue
curl "http://localhost:2170/api/v1/stats/revenue?from=2026-01-01&to=2026-12-31" \
  -H "Authorization: Bearer $TOKEN"

# 10. Upload media
curl -X POST http://localhost:2170/api/v1/media/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@photo.jpg" -F "entity_type=product" -F "entity_id=..."
```

---

## 10. Deliverables

1. **Migration(s):** units, product columns (image_url, sensitivity, unit_id), product_categories pivot, customer code/is_walk_in, order discount, planting_* tables, media, feedback
2. **Handlers:** `/api/v1/*` matching Flutter paths exactly (no trailing slash required)
3. **Adapters:** money string, status int, pagination current_page, order status mapping
4. **OpenAPI:** update `api/openapi.yaml` with mobile contract section
5. **Seed:** owner user, walk-in customer CUS-0001, default units
6. **Docs:** `docs/mobile-api-contract.md` with field mapping tables
7. **Tests:** integration tests for checklist above

---

## 11. Out of scope (OK to differ)

- Flutter local-only: Printer, Hive cache, OrderFilter UI enums
- Go-only features có thể giữ trên `/v1`: expenses, inventory adjustments, profit report, plots admin
- Không yêu cầu đổi Flutter DTOs trong task này
- Category/unit/customer DELETE — Flutter không gọi (soft-delete via status đủ)

---

## 12. Files Flutter reference (đọc khi implement)

```
lib/core/config/api_config.dart
lib/app/data/datasources/api/tinh_tien_api.dart          # All endpoints
lib/app/data/datasources/api/dto/api_dto.dart            # All DTOs
lib/app/data/mappers/api_mappers.dart                    # Write body shapes
lib/app/data/mappers/dto_mappers.dart                  # DTO → domain
lib/app/data/datasources/app_datasource.dart             # ApiDataSource usage
lib/app/data/repositories/app_repository.dart            # Repository interface
lib/app/data/datasources/api/interceptors/api_response_interceptor.dart
```

---

## 13. Acceptance criteria

- [ ] Flutter app login thành công với Go API (đổi API_BASE_URL)
- [ ] Load products, categories, units, customers không lỗi parse JSON
- [ ] Tạo order thành công, inventory trừ đúng
- [ ] Planting schedule CRUD + spray quarantine persist
- [ ] Stats/revenue trả đúng shape cho chart widget
- [ ] Media upload cập nhật product image_url
- [ ] Feedback gửi/đọc được
- [ ] Không breaking change `/v1` routes hiện có (nếu giữ song song)
```

---

## Phụ lục: So sánh nhanh endpoint paths

| Flutter (`/api/v1`) | Go hiện có (`/v1`) | Status |
|---------------------|-------------------|--------|
| POST /auth/login | POST /auth/login | ⚠️ Body khác |
| POST /auth/logout | — | ❌ Thiếu |
| GET /auth/me | — | ❌ Thiếu |
| GET /users | GET /users/ | ⚠️ Shape khác |
| GET /products | GET /products/ | ⚠️ Shape khác |
| GET /categories | GET /categories/ | ⚠️ Shape khác |
| GET /units | — | ❌ Thiếu |
| GET /customers | GET /customers/ | ⚠️ Shape khác |
| GET /orders | GET /orders/ | ⚠️ Shape khác |
| PATCH /orders/{id}/cancel | PATCH /orders/{id}/status | ⚠️ Path + body khác |
| GET /planting-schedules | — | ❌ Thiếu |
| GET /media | — | ❌ Thiếu |
| GET /stats/revenue | GET /reports/sales | ⚠️ Path + shape khác |
| GET /stats/products | GET /reports/products | ⚠️ Path + shape khác |
| GET /stats/customers | GET /reports/receivables | ⚠️ Path + shape khác |
| GET /feedback | — | ❌ Thiếu |

---

*Generated from Flutter client analysis — tinh_tien project.*
