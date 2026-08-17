# Database migrations (code-only)

Schema changes are managed by **versioned Go migrations**, not manual SQL.

## Workflow (lần đầu)

```bash
make docker-up
make migrate          # 001: CREATE tất cả bảng
make migrate-status   # kiểm tra
make seed             # dữ liệu mặc định
make run
```

## How it works

1. Bảng `schema_migrations` ghi migration nào đã chạy.
2. Mỗi migration có **ID cố định**, chỉ chạy **một lần**.
3. **`001_initial_schema`** — cài mới: tạo toàn bộ bảng (CREATE), không rename/alter.
4. **`002+`** — nâng cấp sau này: thêm cột, sửa schema trên DB đã tồn tại.

Registry: [`internal/app/migrations/registry.go`](../internal/app/migrations/registry.go)

## Lần đầu vs nâng cấp

| Tình huống | Làm gì |
|------------|--------|
| **DB mới (dev/prod lần đầu)** | Chỉ cần `001_initial_schema` — AutoMigrate tạo đúng schema từ model |
| **Đã có DB, đổi schema** | Thêm migration `002`, `003`, … — không sửa `001` |

Ví dụ model `settings` dùng cột `setting_key` ngay từ đầu (tránh reserved word `key` của MariaDB).

## Thêm migration nâng cấp (002+)

Khi đã deploy và cần đổi DB:

1. Sửa model trong `internal/domain/...`
2. Thêm entry mới vào `migrations.All()`:

```go
{
    ID:   "002_add_product_sku",
    Name: "add sku column to products",
    Up: func(db *gorm.DB) error {
        exists, err := migrate.ColumnExists(db, "products", "sku")
        if err != nil || exists {
            return err
        }
        return migrate.Exec(db,
            "ALTER TABLE products ADD COLUMN sku VARCHAR(64) NOT NULL DEFAULT ''")
    },
},
```

3. Chạy `make migrate`

**Quy tắc:**

- Không đổi migration ID đã chạy trên production.
- `001` giữ nguyên — chỉ CREATE lúc cài mới.
- Mọi thay đổi sau → ID mới (`002`, `003`, …).

## Reset DB dev (schema cũ / lỗi migrate)

Nếu DB dev bị lỗi từ lần thử trước (cột `key` cũ, index trùng):

```bash
make docker-down
docker volume rm tinh-tien-api_mariadb_data 2>/dev/null || true
make docker-up
make migrate
make seed
```

Hoặc xóa volume tương ứng trong Docker Desktop.

## Current migrations

| ID | Mô tả |
|----|--------|
| `001_initial_schema` | CREATE tất cả bảng (fresh install) |

## Troubleshooting

```bash
make migrate-status
make migrate
make seed
```
