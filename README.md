# Tinh Tien API

Go API for household farm business management (catalog, orders, crop calendar, inventory, statistics).

## Quick start

```bash
make docker-up
make tidy
make migrate    # lần đầu: CREATE tất cả bảng (001)
make seed       # dữ liệu mặc định (owner + settings)
make run
```

API: `http://localhost:2170`  
Docs: `http://localhost:2170/docs` (Swagger UI)  
Spec: `http://localhost:2170/openapi.yaml`  
Health: `GET /healthz`  
Login: `POST /v1/auth/login` with `{"username":"owner","password":"owner123"}`

## Response format

All JSON API responses use the same envelope:

```json
{
  "success": true,
  "message": "products retrieved",
  "data": [],
  "error": null,
  "pagination": null
}
```

List endpoints support `?page=1&page_size=20`. Paginated responses fill the `pagination` object; other endpoints set it to `null`.

Errors (including 404, 401, 403, 500) use the same shape with `success: false`.

## Database (MariaDB)

See [docs/migrations.md](docs/migrations.md) and [docs/ports.md](docs/ports.md).

Docker Compose starts MariaDB 11 with:

- Host: `localhost:2171` (map từ container `3306`)
- Database: `tinh_tien`
- User / password: `tinh_tien` / `tinh_tien`
- Root password: `root`

Default DSN (see `configs/config.yaml`):

```txt
tinh_tien:tinh_tien@tcp(localhost:2171)/tinh_tien?charset=utf8mb4&parseTime=True&loc=UTC
```

## Project layout

Follows a pragmatic subset of [golang-standards/project-layout](https://github.com/golang-standards/project-layout):

- `cmd/api` — application entrypoint
- `cmd/migrate` — run database migrations
- `cmd/seed` — seed default owner and shop settings
- `internal/app` — router, wiring, migrations
- `internal/domain/*` — business modules
- `internal/pkg/*` — shared utilities
- `api/` — OpenAPI spec
- `configs/` — default configuration
- `deployments/` — Docker Compose

## Environment

| Variable      | Description                  |
| ------------- | ---------------------------- |
| `DB_DSN`      | MariaDB connection string    |
| `JWT_SECRET`  | JWT signing secret           |
| `TOKEN_TTL`   | Token lifetime (default `720h`) |
| `HTTP_ADDR`   | Server listen address        |
| `CONFIG_PATH` | Path to YAML config          |
| `APP_TIMEZONE` | Múi giờ thống kê / lọc ngày (mặc định `Asia/Ho_Chi_Minh`). DSN phải `loc=UTC` |
| `PUBLIC_BASE_URL` | URL công khai của API, dùng tạo `file_url` khi upload |
| `UPLOAD_DIR`  | Thư mục lưu file upload (mặc định `uploads`) |
| `OWNER_USERNAME` / `OWNER_PASSWORD` / `OWNER_FULL_NAME` | Tài khoản chủ hộ do `make seed` tạo |
| `APP_ENV`     | `production` → seed bắt buộc có `OWNER_PASSWORD` |

Chuyển dữ liệu từ Firebase: xem [docs/firebase-import.md](docs/firebase-import.md).

## Deploy

VPS (không Docker, nginx + systemd, domain `7b.truongdx.dev`): xem [deploy/README.md](deploy/README.md) — `make deploy`.

## Make targets

- `make migrate` — apply pending DB migrations (code-only, versioned)
- `make migrate-status` — show applied/pending migrations
- `make seed` — seed default data (run once after migrate)
- `make run` — start API (runs migrations on startup). Runs `make db-up` first.
- `make db-up` — make sure the database is up (`scripts/db-up.sh`): opens Docker
  Desktop if needed, starts `DB_CONTAINER` (or docker compose), sets it to
  auto-restart, checks the `DB_DSN` port matches, waits until it accepts
  connections. `migrate`, `migrate-status` and `seed` run it too.
  Skip with `SKIP_DB_UP=1` for a database outside Docker.
- `make stop` — stop the API listening on `HTTP_ADDR` (fixes "port already in use")
- `make test` — run tests
- `make docker-up` — start MariaDB

The API itself also waits for the database for `DB_CONNECT_TIMEOUT` (default
`30s`) and prints a hint with the host/port when it cannot connect.
