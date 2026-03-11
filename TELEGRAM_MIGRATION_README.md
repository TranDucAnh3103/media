# Video Storage Migration: Telegram MTProto

## Tổng quan

Dự án đã được cập nhật để sử dụng **Telegram** làm storage chính cho video thay vì Cloudinary + Mega.nz.

### Lý do chuyển đổi

| Vấn đề cũ                              | Giải pháp mới                        |
| -------------------------------------- | ------------------------------------ |
| Cloudinary giới hạn 100MB/video        | Telegram hỗ trợ file tới 2GB         |
| Mega.nz không hỗ trợ true streaming    | Telegram hỗ trợ HTTP Range requests  |
| Mega tải toàn bộ file trước khi stream | Telegram stream trực tiếp từng chunk |
| Chi phí bandwidth Cloudinary cao       | Telegram miễn phí                    |

---

## Cấu trúc Files mới

```
backend/services/telegram/
├── types.go       # Định nghĩa các struct types
├── errors.go      # Custom error types
├── client.go      # MTProto client initialization
├── uploader.go    # Upload video logic với progress tracking
├── streamer.go    # Streaming với HTTP Range support
├── service.go     # Service wrapper tổng hợp
│
├── # Anti-Ban & Rate-Limit Protection Layer
├── protection.go  # Main protection wrapper kết hợp tất cả modules
├── ratelimit.go   # Token bucket rate limiter (20 uploads/min, 1 msg/sec/user)
├── floodwait.go   # FloodWait error handler với auto-retry
├── throttler.go   # Upload throttler (2-3 concurrent, 3-8s random delay)
├── backoff.go     # Exponential backoff strategy
└── safemode.go    # Safe mode controller với auto-escalation
```

---

## Anti-Ban & Rate-Limit Protection Layer

### Tổng quan

Protection Layer được thiết kế để tuân thủ Telegram Terms of Service và tránh account restrictions:

1. **Rate Limiting** - Giới hạn số requests theo token bucket algorithm
2. **FloodWait Handling** - Tự động xử lý FLOOD_WAIT errors từ Telegram
3. **Upload Throttling** - Giới hạn concurrent uploads và thêm random delays
4. **Exponential Backoff** - Retry với delay tăng dần khi gặp lỗi
5. **Safe Mode** - Tự động escalate throttling khi phát hiện nhiều FloodWait

### Protection Flow

```
User Request
     ↓
[SafeMode Check] ─── Emergency? ──→ Wait or Reject
     ↓
[Rate Limiter] ─── Exceeded? ──→ Wait or Reject
     ↓
[Upload Throttler] ─── Acquire slot + random delay
     ↓
[Apply Safe Mode Multiplier]
     ↓
[Execute with Backoff + FloodWait Handler]
     ↓
Success/Failure + Metrics
```

### Cấu hình mặc định

| Component    | Setting                          | Value         |
| ------------ | -------------------------------- | ------------- |
| Rate Limiter | Max uploads per minute           | 20            |
| Rate Limiter | Max messages per second per user | 1             |
| Rate Limiter | Max global requests per second   | 30            |
| Throttler    | Max concurrent uploads           | 2             |
| Throttler    | Delay range between uploads      | 3-8 seconds   |
| Backoff      | Base delay                       | 1 second      |
| Backoff      | Max delay                        | 5 minutes     |
| Backoff      | Max retries                      | 10            |
| Safe Mode    | FloodWaits to trigger ELEVATED   | 2 in 10 mins  |
| Safe Mode    | FloodWaits to trigger HIGH       | 5 in 10 mins  |
| Safe Mode    | FloodWaits to trigger CRITICAL   | 10 in 10 mins |
| Safe Mode    | FloodWaits to trigger EMERGENCY  | 15 in 10 mins |

### Safe Mode Levels

| Level     | Delay Multiplier | Description                              |
| --------- | ---------------- | ---------------------------------------- |
| NORMAL    | 1.0x             | Normal operation                         |
| ELEVATED  | 1.5x             | Increased delays                         |
| HIGH      | 3.0x             | Significant throttling                   |
| CRITICAL  | 5.0x             | Minimal operations, long waits           |
| EMERGENCY | Blocked          | Stop all operations temporarily (30 min) |

### API để kiểm tra Protection Stats

```go
// Lấy thống kê protection layer
stats := telegramService.GetProtectionStats()
fmt.Printf("Total Requests: %d\n", stats.Metrics.TotalRequests)
fmt.Printf("FloodWaits: %d\n", stats.Metrics.FloodWaitEncountered)
fmt.Printf("Safe Mode: %s\n", stats.SafeMode.LevelString)

// Reset metrics
telegramService.ResetProtectionMetrics()

// Manual safe mode control
telegramService.SetSafeModeLevel(telegram.SafeModeCritical)
telegramService.ResetSafeMode()

// Health check
health := telegramService.HealthCheck()
```

---

## Thiết lập môi trường

### 1. Lấy Telegram API Credentials

1. Truy cập [https://my.telegram.org/apps](https://my.telegram.org/apps)
2. Đăng nhập bằng số điện thoại Telegram
3. Tạo application mới (nếu chưa có)
4. Lưu lại:
   - **API ID** (số)
   - **API Hash** (chuỗi hex 32 ký tự)

### 2. Tạo Private Channel

1. Mở Telegram
2. Tạo channel mới (Settings → New Channel)
3. Chọn **Private Channel**
4. Lấy Channel ID bằng cách:
   - Forward một tin nhắn từ channel đến bot [@userinfobot](https://t.me/userinfobot)
   - Hoặc sử dụng web.telegram.org, mở channel và xem URL (số sau `-100`)
   - Channel ID có dạng: `-1001234567890` (số âm)

### 3. Chọn phương thức xác thực

#### Option A: Bot Token (Khuyến nghị cho production)

1. Tạo bot mới với [@BotFather](https://t.me/BotFather)
2. Lưu token dạng: `123456789:ABCdefGHIjklMNOpqrsTUVwxyz`
3. **Thêm bot làm admin của channel** (quan trọng!)

#### Option B: User Account

1. Sử dụng số điện thoại đăng ký Telegram
2. Cần xử lý OTP code khi đăng nhập lần đầu
3. Nếu có 2FA, cần thêm password

---

## Cấu hình Environment Variables

Tạo file `.env` trong thư mục `backend/`:

```env
# ============ DATABASE ============
MONGO_URI=mongodb+srv://username:password@cluster.mongodb.net/?retryWrites=true&w=majority
MONGO_DB_NAME=media_db

# ============ JWT ============
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production

# ============ CLOUDINARY (Cho ảnh và thumbnail) ============
CLOUDINARY_CLOUD_NAME=your-cloud-name
CLOUDINARY_API_KEY=your-api-key
CLOUDINARY_API_SECRET=your-api-secret

# ============ TELEGRAM (NEW - Cho video storage) ============
# Bắt buộc: API credentials từ my.telegram.org
TELEGRAM_API_ID=12345678
TELEGRAM_API_HASH=0123456789abcdef0123456789abcdef

# Bắt buộc: Private channel ID (số âm, bắt đầu bằng -100)
TELEGRAM_CHANNEL_ID=-1001234567890

# Option A: Bot authentication (khuyến nghị)
TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrsTUVwxyz

# Option B: User authentication
TELEGRAM_PHONE=+84123456789
TELEGRAM_PASSWORD=your-2fa-password-if-enabled

# Tùy chọn: Đường dẫn lưu session
TELEGRAM_SESSION_PATH=./telegram_session

# Tùy chọn: File chứa OTP code cho automated auth
TELEGRAM_CODE_PATH=./telegram_code.txt

# ============ MEGA (DEPRECATED - Backward compatibility) ============
MEGA_EMAIL=your-mega-email@example.com
MEGA_PASSWORD=your-mega-password

# ============ SERVER ============
PORT=8080
CORS_ORIGINS=http://localhost:5173,http://localhost:3000
```

---

## API Endpoints

### Video Streaming

| Endpoint                            | Mô tả                                                  |
| ----------------------------------- | ------------------------------------------------------ |
| `GET /api/videos/stream/:id`        | **MỚI** - Unified streaming (Telegram/Cloudinary/Mega) |
| `GET /api/videos/stream/mega/:hash` | Legacy - Stream từ Mega                                |

### Video Management

| Endpoint                              | Mô tả                                |
| ------------------------------------- | ------------------------------------ |
| `POST /api/videos/upload`             | Upload video (tự động dùng Telegram) |
| `GET /api/videos/upload/progress/:id` | Lấy tiến trình upload                |
| `DELETE /api/videos/:id`              | Xóa video (xóa cả trên storage)      |

---

## Database Schema Update

### Video Model - Fields mới

```go
type Video struct {
    // ... existing fields ...

    // New Telegram fields
    StorageProvider    string  // "telegram", "cloudinary", "mega"
    TelegramChannelID  int64   // Channel ID
    TelegramMessageID  int     // Message ID chứa video
    TelegramFileID     string  // File ID trên Telegram
    MimeType           string  // Video MIME type
    Width              int     // Video width
    Height             int     // Video height
}
```

### Helper Method

```go
// Tự động lấy stream URL dựa trên storage provider
video.GetStreamURL() // → "/api/videos/stream/{id}"
```

---

## Upload Flow mới

```
User Upload Video
       ↓
Backend nhận file
       ↓
Tạo record MongoDB (status: processing)
       ↓
[Background Goroutine]
       ↓
Upload lên Telegram Channel
       ↓
Lưu message_id, file_id vào MongoDB
       ↓
Cập nhật status: ready
```

---

## Streaming Flow mới

```
Client request: GET /api/videos/stream/:id
                Range: bytes=0-1048576
       ↓
Backend parse Range header
       ↓
Fetch chunk từ Telegram via MTProto
       ↓
HTTP 206 Partial Content
Content-Range: bytes 0-1048576/FILE_SIZE
       ↓
Client nhận stream
```

---

## Backward Compatibility

- Video cũ trên **Cloudinary** → Redirect tới URL gốc
- Video cũ trên **Mega** → Stream qua `/api/videos/stream/mega/:hash`
- Video mới → Stream qua Telegram

---

## Chạy ứng dụng

### Backend

```bash
cd backend

# Install dependencies
go mod tidy

# Run
go run .
```

### Frontend

```bash
cd frontend

# Install dependencies
npm install

# Run dev server
npm run dev
```

---

## Troubleshooting

### 1. Lỗi "Telegram client not connected"

- Kiểm tra `TELEGRAM_API_ID` và `TELEGRAM_API_HASH`
- Đảm bảo bot đã được add làm admin của channel

### 2. Lỗi "Channel not found"

- Kiểm tra `TELEGRAM_CHANNEL_ID` có đúng format (số âm, bắt đầu -100)
- Đảm bảo bot/user có quyền truy cập channel

### 3. Lỗi "Authentication required"

- Với bot: Kiểm tra `TELEGRAM_BOT_TOKEN`
- Với user: Kiểm tra số điện thoại và xử lý OTP

### 4. Upload thất bại "File too large"

- Telegram giới hạn file tối đa 2GB
- Kiểm tra kích thước file trước khi upload

### 5. Video không phát được

- Kiểm tra `storage_provider` trong database
- Đảm bảo endpoint streaming đúng

---

## Dependencies

### Go modules mới

```
github.com/gotd/td v0.91.0  # Telegram MTProto library
```

### Cài đặt

```bash
go get github.com/gotd/td@v0.91.0
go mod tidy
```

---

## Risks & Limitations

| Risk                   | Mitigation                          |
| ---------------------- | ----------------------------------- |
| Telegram rate limiting | Implement retry với backoff         |
| Connection instability | Auto-reconnect mechanism            |
| 2GB file limit         | Compress/transcode trước khi upload |
| Account ban            | Sử dụng bot riêng, tuân thủ TOS     |

---

## Future Improvements

- [ ] CDN cache layer (Cloudflare)
- [ ] Video transcoding pipeline
- [ ] HLS adaptive streaming
- [ ] Multi-account upload balancing
- [ ] Thumbnail extraction từ video

---

## Contact

Nếu có vấn đề, vui lòng tạo issue trên repository.
