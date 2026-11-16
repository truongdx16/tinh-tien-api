# Nhập dữ liệu từ Firebase (production)

Dữ liệu được export và chuyển đổi bởi tool trong repo Flutter: `tinh_tien/tools/tinh_tien_admin`
(`npm run export:firebase` rồi `npm run transform:go`). Kết quả là file `export/go/import.sql`, ghi trực tiếp
vào các bảng do `cmd/migrate` tạo.

## Quy trình

```bash
# 0. Biến môi trường production
export DB_DSN='user:pass@tcp(db-host:3306)/tinh_tien?charset=utf8mb4&parseTime=True&loc=UTC'   # loc=UTC bắt buộc
export JWT_SECRET='<chuỗi ngẫu nhiên dài>'
export APP_TIMEZONE=Asia/Ho_Chi_Minh
export PUBLIC_BASE_URL=https://api.<domain>
export APP_ENV=production

# 1. Tạo bảng (001 → 003)
go run ./cmd/migrate

# 2. Nhập dữ liệu Firebase (chạy lại được, upsert theo id)
mysql -h db-host -u user -p tinh_tien < import.sql

# 3. Tài khoản chủ hộ + dữ liệu mặc định (Firebase không xuất được mật khẩu)
OWNER_USERNAME=<email đăng nhập trên app> OWNER_PASSWORD='<mật khẩu mới>' OWNER_FULL_NAME='Chủ hộ' go run ./cmd/seed

# 4. Chạy API
go run ./cmd/api
```

Lỡ chạy `seed` trước `import.sql` vẫn an toàn: import sẽ gộp khách vãng lai `CUS-0001` và các đơn vị mặc định trùng tên.

## Kiểm tra sau khi nhập

```sql
SELECT COUNT(*), SUM(total) FROM orders WHERE status <> 'cancelled';   -- so với report.json (orders.unique_orders, revenue_total)
SELECT COUNT(*) FROM customers WHERE code = 'CUS-0001';                -- phải = 1
```

## Quy ước dữ liệu

| Firebase / Flutter | Go |
| --- | --- |
| Đơn lịch sử (đã bán) | `orders.status = 'delivered'` (Flutter 2), `allow_backorder = 1`, `paid_amount = total` |
| Số đơn cũ `ORD-yyyyMMdd-HHmmss` | `orders.note` |
| Khách vãng lai `id = "CUS-0001"` trên app | `customers.code = 'CUS-0001'`, id UUID; API tự đổi qua lại |
| Sản phẩm chỉ còn trong đơn cũ | `products.active = 0` (để thống kê theo `product_id` vẫn đúng) |
| Ảnh Firebase Storage | `image_url` vẫn trỏ Firebase (bucket hết billing → 402). Chép file vào `UPLOAD_DIR` và chạy transform với `--media-base-url=$PUBLIC_BASE_URL/uploads` để đổi URL |

## Hành vi API mobile đã chỉnh cho production

- `POST /orders`: tạo đơn `delivered` (bán tại quầy), lưu `discount` và `paid_amount` (công nợ = `balance_due`), không kiểm tra tồn kho, nhận `customer_id = "CUS-0001"`.
- `PATCH /orders/{id}/cancel`: hủy được cả đơn đã hoàn thành; chỉ hoàn kho nếu đơn đã từng trừ kho.
- `GET /orders`, `/stats/*`: `from`/`to` là ngày theo `APP_TIMEZONE`, bao gồm trọn ngày `to`; doanh thu theo ngày nhóm theo giờ VN.
- `GET /products`: `sales_quantity`/`revenue` tính từ `order_items`, `quantity` từ `balances`; lọc `category_id` theo bảng pivot.
- `/planting-schedules`: nhận số hoặc chuỗi; trường rỗng trả `null` (Flutter hiểu `""` là ngày hôm nay).
- `/feedback`: `id` kiểu int.
