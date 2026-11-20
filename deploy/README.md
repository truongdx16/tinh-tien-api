# Deploy Tinh Tien API lên VPS — `7b.truongdx.dev`

Không dùng Docker. MariaDB đã có sẵn trên VPS. Nginx đứng trước (HTTPS Let's Encrypt).

```text
App Flutter ──https──► Nginx :443 (7b.truongdx.dev)
                          ├─ /uploads/*  → đọc thẳng /var/www/tinh-tien-api/uploads
                          └─ /*          → Go API 127.0.0.1:2170 (systemd: tinh-tien-api)
                                              └─ MariaDB 127.0.0.1:3306 (có sẵn)
```

**Nguyên tắc:** build ở máy local (Go → 1 file chạy Linux), rsync lên VPS, đổi symlink `current`, restart service. VPS không cần Go, không cần git clone.

```text
/var/www/tinh-tien-api/
├── current -> releases/20261118-132211     # bản đang chạy
├── releases/<thời-gian>/                   # giữ 5 bản gần nhất (rollback)
│   ├── bin/{api,migrate,seed}
│   ├── configs/config.yaml
│   ├── api/openapi.yaml                    # /docs
│   └── REVISION                            # git commit
├── .env                                    # từ .env.production (quyền 600)
├── .env.bak-<thời-gian>                    # 5 bản .env cũ
└── uploads/                                # ảnh sản phẩm — KHÔNG bị deploy ghi đè
```

## 1. Chuẩn bị (làm 1 lần)

### Trên VPS (Ubuntu/Debian)

- Trỏ DNS: bản ghi **A** `7b.truongdx.dev` → IP VPS (đợi lan truyền: `dig +short 7b.truongdx.dev`).
- Cài nginx + certbot nếu chưa có: `apt update && apt install -y nginx certbot`
- MariaDB đã chạy, đăng nhập được bằng `mariadb -uroot` khi là root (mặc định trên Ubuntu).
- Mở cổng 80, 443 (`ufw allow 'Nginx Full'` nếu dùng ufw). **Không** mở 2170 (API chỉ nghe 127.0.0.1).

### Trên máy local

```bash
cd tinh-tien-api
ssh-copy-id root@<IP-VPS>                      # đăng nhập SSH bằng key

cp scripts/deploy.config.example.env scripts/deploy.config.env
#   → điền VPS_HOST, CERTBOT_EMAIL (TARGET_ARCH=arm64 nếu VPS ARM)

cp .env.production.example .env.production
#   → DB_DSN: user/mật khẩu/tên DB cho production
#   → JWT_SECRET: openssl rand -hex 32
#   → OWNER_USERNAME / OWNER_PASSWORD: tài khoản chủ hộ đầu tiên
```

Hai file trên đã nằm trong `.gitignore` — không commit.

## 2. Lần đầu

```bash
./scripts/deploy.sh setup     # user `tinhtien`, thư mục, nginx, chứng chỉ HTTPS, systemd
./scripts/deploy.sh db-init   # tạo database + user trong MariaDB có sẵn (bỏ qua nếu đã có)
./scripts/deploy.sh           # build + deploy; API tự tạo bảng khi khởi động
./scripts/deploy.sh seed      # tạo tài khoản chủ hộ (OWNER_* trong .env.production)
```

Sau `seed`: xoá `OWNER_PASSWORD` khỏi `.env.production` và deploy lại; đổi mật khẩu trong app (Quản lý › Đổi mật khẩu).

Kiểm tra: `https://7b.truongdx.dev/healthz`, tài liệu API `https://7b.truongdx.dev/docs`.

> **Chuyển dữ liệu local lên:** `mysqldump` DB local rồi import vào DB production **trước** lần deploy đầu, ví dụ
> `mysqldump -h127.0.0.1 -P3307 -uroot -p tinh_tien_go | ssh root@<IP> "mariadb -uroot tinh_tien"`.
> Ảnh đã upload ở local nằm trong `uploads/` — rsync vào `/var/www/tinh-tien-api/uploads/` rồi
> `chown -R tinhtien:tinhtien /var/www/tinh-tien-api/uploads`.

## 3. Các lần sau

```bash
./scripts/deploy.sh              # = make deploy
```

Script sẽ: kiểm tra `.env.production` → chạy `go test ./...` → build linux → upload bản mới → upload `.env` (nếu đổi, có backup) → đổi `current` → restart → chờ `/healthz`. API lỗi khi khởi động thì script in log và gợi ý rollback.

| Lệnh | Việc |
|---|---|
| `./scripts/deploy.sh` | Deploy |
| `./scripts/deploy.sh rollback` | Quay về bản trước (không đụng DB, `.env`, ảnh) |
| `./scripts/deploy.sh status` | Trạng thái service, bản đang chạy, health |
| `./scripts/deploy.sh logs` | Xem log trực tiếp (`journalctl -f`) |
| `DRY_RUN=true ./scripts/deploy.sh` | Xem thử, không thay đổi gì |
| `RUN_TESTS=false ./scripts/deploy.sh` | Bỏ qua test (gấp) |

**Đổi cấu hình** (DB, URL…): sửa `.env.production` ở local rồi deploy — đừng sửa `.env` trên VPS (lần deploy sau sẽ ghi đè).
Đổi `JWT_SECRET` làm mọi người bị đăng xuất, nên script dừng lại; muốn đổi thật: `FORCE_JWT_CHANGE=true ./scripts/deploy.sh`.

## 4. App Flutter

Bản **release** tự dùng `https://7b.truongdx.dev/api/v1` (xem `lib/core/config/api_config.dart`):

```bash
cd tinh_tien
fvm flutter build apk --release        # Android
fvm flutter build ipa --release        # iOS
```

Bản debug vẫn gọi API local; muốn debug với API production: chọn cấu hình **"tinh_tien (production API)"** trong VS Code, hoặc
`fvm flutter run --dart-define=API_BASE_URL=https://7b.truongdx.dev/api/v1`.

## 4b. Bản web (cùng domain với API)

App Flutter bản web chạy ở `https://7b.truongdx.dev/`, cùng nguồn gốc với API nên không cần CORS. Nginx trả file tĩnh ở `/var/www/tinh-tien-web`; `/api`, `/v1`, `/docs`, `/healthz`, `/uploads` vẫn đi tới API.

```bash
./scripts/deploy.sh setup   # một lần (và mỗi khi đổi deploy/nginx): cài lại site nginx
./scripts/deploy.sh web     # build Flutter web (../tinh_tien, đổi bằng WEB_APP_DIR) + upload
```

- In hoá đơn trên web đi qua hộp thoại in của trình duyệt (trình duyệt không mở được cổng TCP 9100). Chọn khổ giấy 58/80 mm ở Cài đặt › Máy in.
- Không có trên web: thông báo đẩy, lắc máy mở báo cáo, tự cập nhật APK, tìm máy in trong mạng.

## 5. Bảo mật đã có

- API chỉ nghe `127.0.0.1:2170`; chạy bằng user `tinhtien` (không phải root), systemd chặn ghi ngoài `uploads/`.
- `APP_ENV=production` ⇒ API **từ chối khởi động** nếu `JWT_SECRET` yếu/mặc định hoặc `PUBLIC_BASE_URL` không phải https.
- Nginx: HTTPS + HSTS, giới hạn đăng nhập 10 lần/phút/IP, API 20 req/s/IP, upload tối đa 32 MB.
- `.env` trên VPS quyền 600 (root). Mật khẩu DB ở `db-init` đi qua stdin, không hiện trên dòng lệnh.
- Chứng chỉ tự gia hạn (certbot timer) và nginx tự reload sau gia hạn.

## 6. Xử lý sự cố

| Triệu chứng | Kiểm tra |
|---|---|
| `Cannot SSH` | `ssh-copy-id root@<IP>`; đúng `VPS_PORT`? |
| `setup` lỗi certbot | DNS đã trỏ đúng IP chưa (`dig +short 7b.truongdx.dev`); cổng 80 mở chưa |
| Deploy báo API không lên | `./scripts/deploy.sh logs` — thường là `DB_DSN` sai hoặc DB chưa tạo (`db-init`) |
| 502 Bad Gateway | `systemctl status tinh-tien-api`; `curl http://127.0.0.1:2170/healthz` trên VPS |
| Ảnh upload không hiện | `PUBLIC_BASE_URL=https://7b.truongdx.dev`; quyền `uploads/` thuộc `tinhtien` |
| 413 khi upload | Ảnh > 32 MB (giới hạn của API và nginx) |

Log nginx: `/var/log/nginx/7b.truongdx.dev.{access,error}.log`. Log deploy ở máy local: `deploy/logs/`.
