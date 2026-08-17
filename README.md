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

API: `http://localhost:8080`  
Docs: `http://localhost:8080/docs` (Swagger UI)  
Spec: `http://localhost:8080/openapi.yaml`  
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

See [docs/migrations.md](docs/migrations.md) for upgrading schema through code (versioned migrations).

Docker Compose starts MariaDB 11 with:

- Host: `localhost:3306`
- Database: `tinh_tien`
- User / password: `tinh_tien` / `tinh_tien`
- Root password: `root`

Default DSN (see `configs/config.yaml`):

```txt
tinh_tien:tinh_tien@tcp(localhost:3306)/tinh_tien?charset=utf8mb4&parseTime=True&loc=Local
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
| `HTTP_ADDR`   | Server listen address        |
| `CONFIG_PATH` | Path to YAML config          |

## Make targets

- `make migrate` — apply pending DB migrations (code-only, versioned)
- `make migrate-status` — show applied/pending migrations
- `make seed` — seed default data (run once after migrate)
- `make run` — start API (runs migrations on startup)
- `make test` — run tests
- `make docker-up` — start MariaDB
