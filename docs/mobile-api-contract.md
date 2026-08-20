# Mobile API Contract — Flutter client alignment

Base URL: `http://<host>:2170/api/v1`  
Auth: `Authorization: Bearer <token>`

## Auth

| Method | Path           | Body / Response                                    |
| ------ | -------------- | -------------------------------------------------- |
| POST   | `/auth/login`  | `{email, password, device_name}` → `{token, user}` |
| POST   | `/auth/logout` | — → void (no-op; client deletes token)             |
| GET    | `/auth/me`     | — → `UserDto`                                      |
| GET    | `/users`       | — → `UserDto[]`                                    |

**UserDto**: `{id, name, email, store_id, role, status: int}`  
Note: `email` field maps to `username` in DB (login by username).

## Catalog

| Method | Path               | Notes                                                                            |
| ------ | ------------------ | -------------------------------------------------------------------------------- |
| GET    | `/categories`      | Returns `CategoryDto[]`                                                          |
| POST   | `/categories`      | `{name, status}`                                                                 |
| PUT    | `/categories/{id}` | `{name?, status?}`                                                               |
| GET    | `/units`           | Returns `UnitDto[]`                                                              |
| POST   | `/units`           | `{name, slug?, status}`                                                          |
| PUT    | `/units/{id}`      |                                                                                  |
| GET    | `/products`        | `?status=1&search=&category_id=&per_page=20` → paginated                         |
| POST   | `/products`        | `{name, image_url, price, quantity, sensitivity, unit_id, status, category_ids}` |
| PUT    | `/products/{id}`   | Same body                                                                        |
| DELETE | `/products/{id}`   |                                                                                  |

**CategoryDto**: `{id, name, status: int}`  
**UnitDto**: `{id, name, slug, status: int}`  
**ProductDto**: `{id, name, image_url, price, quantity, sales_quantity, revenue, sensitivity, status, unit, categories[]}`

- `price`, `quantity`, `revenue` are **decimal strings**
- `status` is **int** (1=active, 0=inactive)

Paginated response shape:

```json
{"success": true, "data": {"data": [...], "current_page": 1, "total": 42, "per_page": 20}}
```

## Customers

| Method | Path              | Notes                                                 |
| ------ | ----------------- | ----------------------------------------------------- |
| GET    | `/customers`      | `?search=&per_page=20` → paginated                    |
| POST   | `/customers`      | `{code?, name, phone?, address?, status, is_walk_in}` |
| PUT    | `/customers/{id}` |                                                       |

**CustomerDto**: `{id, code, name, phone, address, status: int, is_walk_in: bool}`  
Walk-in convention: `code == "CUS-0001"` seeded automatically.

## Orders

| Method | Path                  | Notes                                                    |
| ------ | --------------------- | -------------------------------------------------------- |
| GET    | `/orders`             | `?from=YYYY-MM-DD&to=YYYY-MM-DD&status=0` → `OrderDto[]` |
| POST   | `/orders`             | create                                                   |
| GET    | `/orders/{id}`        |                                                          |
| PATCH  | `/orders/{id}/cancel` | cancels order                                            |

**OrderDto**: `{id, customer_id, discount, revenue, paid_amount, status: int, created_at, updated_at, customer, items[]}`  
**Item**: `{product_id, product_name, unit_name, unit_price, quantity, line_total}` — all money fields are strings  
**Status mapping**: 0=draft/pending, 1=confirmed/processing, 2=delivered/completed, 3=cancelled  
**Create body**: `{customer_id?, discount, paid_amount, items: [{product_id, quantity}]}`

## Planting Schedules

| Method | Path                       |
| ------ | -------------------------- |
| GET    | `/planting-schedules`      |
| POST   | `/planting-schedules`      |
| GET    | `/planting-schedules/{id}` |
| PUT    | `/planting-schedules/{id}` |
| DELETE | `/planting-schedules/{id}` |

Nested arrays: `spray_tasks`, `fertilize_tasks`, `planting_tasks` — replaced on update.  
Key field: `quarantine_date` / `quarantine_days` in `SprayTask` — Flutter uses this to block harvest.

## Media

| Method | Path            | Notes                                           |
| ------ | --------------- | ----------------------------------------------- |
| GET    | `/media`        | `MediaDto[]`                                    |
| POST   | `/media/upload` | multipart: `file`, `entity_type?`, `entity_id?` |

Uploaded files served at `/uploads/<filename>`.

## Stats

| Method | Path               | Params       |
| ------ | ------------------ | ------------ |
| GET    | `/stats/revenue`   | `from`, `to` |
| GET    | `/stats/products`  | `limit?`     |
| GET    | `/stats/customers` | `from`, `to` |

## Feedback

| Method | Path        |
| ------ | ----------- | -------------------------------- |
| GET    | `/feedback` |
| POST   | `/feedback` | `{content, rating?, full_name?}` |

## Field mapping table

| Flutter field     | Go field           | Conversion         |
| ----------------- | ------------------ | ------------------ |
| `email` (login)   | `username`         | direct map         |
| `name` (user)     | `full_name`        | —                  |
| `status: int`     | `active: bool`     | 1↔true, 0↔false    |
| `price` string    | `sell_price int64` | int64 → string     |
| `revenue` string  | `total int64`      | int64 → string     |
| `discount` string | `discount int64`   | int64 → string     |
| `current_page`    | `page`             | pagination adapter |

## Existing `/v1` routes

All original `/v1` routes are unchanged and still work for admin/web clients.
