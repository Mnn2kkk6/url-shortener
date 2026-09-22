# URL Shortener (Trợ lý rút gọn link)

Dịch vụ rút gọn URL viết bằng Go: nhận một URL dài, sinh ra một mã ngắn
duy nhất, và chuyển hướng (HTTP redirect) người dùng về link gốc khi họ
truy cập mã đó. Có kèm giao diện web đơn giản để nhập link trực tiếp
trên trình duyệt, không bắt buộc phải dùng curl hay Postman.

## Kiến trúc

- **PostgreSQL**: nguồn dữ liệu chính (source of truth), lưu ánh xạ
  `code -> long_url` dạng key-value trong bảng `urls`.
- **Redis**: lớp cache đọc (read-through cache) đặt trước PostgreSQL.
  Khi redirect, server thử đọc Redis trước; nếu cache miss mới đọc
  PostgreSQL rồi ghi lại vào cache. Vì số lượt đọc (click vào link ngắn)
  thường nhiều gấp nhiều lần số lượt ghi (tạo link mới), cache giúp giảm
  tải cho PostgreSQL và giảm độ trễ redirect.
- **net/http (Go 1.22+)**: dùng `http.ServeMux` có sẵn với cú pháp định
  tuyến theo method + path pattern (`GET /{code}`), không cần thêm
  framework ngoài.
- **Giao diện web (`web/index.html`)**: một trang HTML/CSS/JS thuần,
  không cần build, được `main.go` phục vụ tại `GET /` bằng
  `http.ServeFile`. Trang gọi thẳng tới `POST /api/shorten` bằng
  `fetch()` và hiển thị kết quả kèm nút sao chép.

```
Trình duyệt --GET /--> web/index.html (form nhập link)
                              |
                              v (fetch JS)
Client --POST /api/shorten--> Handler --sinh mã--> Postgres (lưu)
                                              \--> Redis (làm nóng cache)

Client --GET /{code}--> Handler --đọc--> Redis (cache hit, nhanh)
                                    \--miss--> Postgres --> ghi lại Redis
                              --> HTTP 301 Redirect về URL gốc
```

## Cấu trúc thư mục

```
urlshortener/
├── cmd/server/main.go          # điểm khởi chạy, wiring dependencies
├── internal/config/            # đọc cấu hình từ biến môi trường
├── internal/shortener/         # sinh mã ngắn base62 ngẫu nhiên
├── internal/store/              # PostgresStore + CachedStore (Redis)
├── internal/handler/           # HTTP handlers (shorten, redirect)
├── web/index.html               # giao diện web (form nhập link)
├── migrations/001_init.sql     # schema PostgreSQL
├── docker-compose.yml          # Postgres + Redis cho local dev
├── .env.example                # mẫu biến môi trường
├── requests.http                # test API nhanh trong VSCode
└── .vscode/                    # cấu hình debug/format cho VSCode
```

> ⚠️ **Quan trọng**: `web/index.html` phải nằm ở **thư mục gốc** của
> project (ngang hàng với `go.mod`), **không phải** bên trong `.vscode`
> hay bất kỳ thư mục con nào khác. Server đọc file này bằng đường dẫn
> tương đối `web/index.html`, tính từ nơi bạn gõ lệnh `go run`. Kiểm tra
> nhanh bằng lệnh: `Test-Path "web\index.html"` (PowerShell) phải trả
> về `True`.

## Yêu cầu

- Go 1.22 trở lên
- Docker + Docker Compose (để chạy Postgres/Redis local), hoặc tự cài
  Postgres/Redis nếu muốn
- VSCode với extension **Go** (golang.go) — cài khi VSCode gợi ý mở
  project lần đầu

## Chạy thử ở máy local

```bash
# 1. Copy file cấu hình mẫu
cp .env.example .env

# 2. Bật Postgres + Redis bằng Docker (tự chạy migration lúc khởi tạo)
docker compose up -d

# 3. Tải dependency Go (bắt buộc, vì go.mod chưa có go.sum)
go mod tidy

# 4. Chạy server
go run ./cmd/server
# hoặc: make run
```

Server chạy tại `http://localhost:8080`. Mở file `requests.http` trong
VSCode (cài extension "REST Client") để bấm gửi request trực tiếp,
hoặc dùng curl:

```bash
# Tạo link rút gọn
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.anthropic.com/claude"}'

# Kết quả mẫu:
# {"code":"aZ3kD9","short_url":"http://localhost:8080/aZ3kD9","long_url":"https://www.anthropic.com/claude"}

# Truy cập link rút gọn -> chuyển hướng về URL gốc
curl -i http://localhost:8080/aZ3kD9
```

### Hoặc dùng giao diện web (đơn giản nhất)

Mở trình duyệt vào `http://localhost:8080`, dán link dài vào ô nhập,
bấm **"Rút gọn"** — link rút gọn hiện ra ngay kèm nút sao chép, không
cần dùng curl hay `requests.http` nữa.

## Debug trong VSCode

Nhấn `F5` (hoặc vào tab Run and Debug) và chọn cấu hình
**"Debug URL Shortener Server"** — đã cấu hình sẵn trong
`.vscode/launch.json`, tự nạp biến môi trường từ `.env`.

## Chạy test

```bash
go test ./... -v
# hoặc: make test
```

## Các điểm kỹ thuật đáng chú ý (để ôn lại khi học)

- **Thiết kế key-value**: bảng `urls` chỉ cần khóa chính `code`, tra cứu
  O(1)/O(log n) theo index — đúng bản chất bài toán key-value.
- **Sinh mã duy nhất**: sinh ngẫu nhiên base62 bằng `crypto/rand`, kiểm
  tra trùng bằng `Exists()` trước khi ghi, thử lại tối đa 5 lần
  (`internal/handler/handler.go`).
- **HTTP redirect**: dùng `http.Redirect` với mã `301 Moved Permanently`
  — phù hợp vì URL rút gọn trỏ cố định về một đích, giúp trình duyệt/CDN
  cache lại redirect này.
- **Tối ưu hiệu năng đọc**: chiến lược cache-aside với Redis
  (`internal/store/cache.go`), cập nhật `hits` bất đồng bộ bằng
  goroutine để không chặn đường redirect chính.
- **Graceful shutdown**: `main.go` lắng nghe `SIGINT`/`SIGTERM` và đóng
  server đúng cách, tránh request đang xử lý bị cắt ngang đột ngột.

## Đẩy dự án lên GitHub

```bash
cd urlshortener
git init
git add .
git commit -m "feat: URL shortener voi PostgreSQL + Redis cache"

# Tạo repo rỗng trên GitHub trước (không tick "Initialize with README"),
# sau đó:
git branch -M main
git remote add origin https://github.com/<ten-user-cua-ban>/urlshortener.git
git push -u origin main
```

> Lưu ý: file `.env` đã được thêm vào `.gitignore` nên sẽ không bị commit
> lên GitHub — chỉ `.env.example` (không chứa bí mật thật) mới được đẩy lên.

## Hướng phát triển tiếp theo (gợi ý)

- Thêm rate limiting để chống spam tạo link.
- Thêm endpoint thống kê số lượt click theo mã (`GET /api/stats/{code}`).
- Cho phép người dùng đặt mã tùy chỉnh (custom alias).
- Thêm TTL/hết hạn cho từng link.
- Viết Dockerfile để đóng gói chính ứng dụng Go (không chỉ Postgres/Redis).