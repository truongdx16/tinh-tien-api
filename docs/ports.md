# Dev ports (số nhà 21, kênh 7)

Dải port dev tránh trùng với service mặc định (8080, 3306).

| Port | Service | Ghi chú |
|------|---------|---------|
| **2170** | API HTTP | `21` + kênh `7` → `:2170` |
| **2171** | MariaDB (host) | Map vào container `3306` |
| **2172** | Swagger UI (docker) | `make openapi-ui` |

## Override qua env

```bash
HTTP_ADDR=:2170
DB_DSN=tinh_tien:tinh_tien@tcp(localhost:2171)/tinh_tien?charset=utf8mb4&parseTime=True&loc=Local
```

## Thêm service sau này

Giữ pattern `217X` (2173, 2174, …) cho redis, worker, frontend dev, v.v.
