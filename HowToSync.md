# Hướng dẫn đồng bộ video từ Telegram Channel

## Cách sync tin nhắn có video hoạt động

### Luồng async (khuyến nghị)

1. **Bắt đầu sync**: `POST /api/videos/telegram/sync/start` (cần đăng nhập)

   - Response ngay HTTP 202:
   ```json
   { "started": true, "sync_id": "sync_1234567890" }
   ```

2. **Kiểm tra tiến độ**: `GET /api/videos/telegram/sync/status`
   ```json
   {
     "is_running": true,
     "sync_id": "sync_1234567890",
     "current_progress": 120,
     "total": 500,
     "new_count": 40,
     "last_error": null
   }
   ```
   - Poll mỗi 2–3 giây cho đến khi `is_running: false`.

3. **Backend chạy sync trong goroutine**:
   - Quét channel → lọc video → tải thumbnail → upload Cloudinary → lưu MongoDB.

### Luồng đồng bộ (legacy)

`POST /api/videos/telegram/sync` – vẫn hỗ trợ, chạy toàn bộ sync trong request (dễ timeout nếu channel nhiều video).

### Dữ liệu được lưu

| Trường | Nguồn |
|--------|--------|
| Title | Caption hoặc tên file hoặc "Video {msgID}" |
| Description | Caption từ tin nhắn |
| Thumbnail | Cloudinary URL (từ thumbnail của document Telegram) |
| Duration, FileSize, Width, Height | Metadata từ Telegram |
| TelegramMessageID, TelegramFileID | Để stream sau này |
| StorageProvider | `"telegram"` |

### Điều kiện thumbnail có

- Video trên Telegram có thumbnail (`doc.Thumbs` không rỗng)
- Cloudinary đã cấu hình (env `CLOUDINARY_*`)
- Kết nối Telegram ổn định

### Cập nhật thumbnail sau khi không có (lazy generation)

Video vẫn được sync và stream bình thường kể cả khi **không có** `doc.Thumbs` trên Telegram. Khi user mở video để xem lần đầu, backend có thể tự động tạo thumbnail (frame đầu) và lưu vào record.

**Luồng đề xuất:**

1. **Endpoint**: `GET /api/videos/:id/thumbnail`
   - Nếu có thumbnail → redirect 302 đến URL Cloudinary
   - Nếu chưa có → tạo thumbnail → cập nhật DB → redirect đến URL mới

2. **Cách tạo thumbnail:**
   - Tải chunk đầu tiên của video từ Telegram (vài trăm KB – 1 MB)
   - Ghi ra file tạm
   - FFmpeg: `ffmpeg -i temp.mp4 -vframes 1 -ss 0.5 -q:v 2 thumb.jpg`
   - Upload lên Cloudinary
   - Cập nhật `Video` trong MongoDB: `{ $set: { thumbnail: cloudinaryURL } }`
   - Xóa file tạm

3. **Lưu ý:**
   - Chỉ áp dụng cho video `StorageProvider = "telegram"`
   - Cần FFmpeg trên server
   - Cần Cloudinary đã cấu hình
   - Có thể chạy trong goroutine để không block stream

**Frontend:** dùng `GET /api/videos/:id/thumbnail` làm `src` cho `<img>` hoặc poster của `<video>`. Lần đầu request sẽ trigger tạo thumbnail nếu chưa có.

---

## API Endpoints

| Method | Endpoint | Auth | Mô tả |
|--------|----------|------|------|
| POST | `/api/videos/telegram/sync/start` | Yes | Bắt đầu sync nền, trả 202 ngay |
| GET | `/api/videos/telegram/sync/status` | No | Trạng thái sync (is_running, current_progress, total, new_count, last_error) |
| POST | `/api/videos/telegram/sync` | Yes | Sync đồng bộ (legacy) |
| GET | `/api/videos/:id/thumbnail` | No | Lấy thumbnail; nếu chưa có (video Telegram) thì tạo từ frame đầu rồi redirect |

## Frontend gợi ý

1. Gọi `POST /telegram/sync/start` → hiển thị "Đang đồng bộ..."
2. Poll `GET /telegram/sync/status` mỗi 2–3 giây
3. Khi `is_running: false` → thông báo hoàn thành, refresh danh sách video
