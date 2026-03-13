# Telegram Channel Video Sync Feature

## Tổng quan

Tính năng này cho phép đồng bộ video từ kênh Telegram vào hệ thống, sử dụng Telegram làm nơi lưu trữ video và stream video qua backend đến client.

## Kiến trúc

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Telegram       │────▶│  Backend API    │────▶│  Frontend       │
│  Channel        │     │  (Go/Fiber)     │     │  (React)        │
│  (Video Storage)│◀────│  + MongoDB      │◀────│                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

## Các file đã tạo/sửa đổi

### Backend

#### 1. Model mới: `backend/models/telegram_channel_video.go`

```go
type TelegramChannelVideo struct {
    ID                   primitive.ObjectID
    TelegramMessageID    int       // ID tin nhắn trên Telegram
    TelegramGroupedID    int64     // ID album (nếu video thuộc album)
    TelegramChannelID    int64     // ID kênh Telegram
    TelegramFileID       string    // File ID trên Telegram
    TelegramFileRef      []byte    // File reference (có thể hết hạn)
    TelegramAccessHash   int64     // Access hash của document
    Caption              string    // Caption của message
    Duration             int       // Thời lượng video (giây)
    FileSize             int64     // Kích thước file (bytes)
    Width, Height        int       // Độ phân giải
    MimeType             string    // MIME type
    FileName             string    // Tên file gốc
    TelegramMessageDate  time.Time // Ngày gửi trên Telegram
    SyncedAt             time.Time // Ngày đồng bộ
    IsPublished          bool      // Đã publish thành video chính chưa
    PublishedVideoID     primitive.ObjectID // ID video đã publish
    HasSpoiler           bool
    SupportsStreaming    bool
}
```

#### 2. Service mới: `backend/services/telegram/scanner.go`

**ChannelScanner** - Quét kênh Telegram để lấy video:

- `ScanChannel(ctx, opts)` - Quét channel và trả về danh sách video metadata
- `FetchSingleMessage(ctx, channelID, accessHash, messageID)` - Lấy thông tin 1 message
- `RefreshFileReference(ctx, channelID, accessHash, messageID)` - Làm mới file reference

**Cấu trúc dữ liệu:**

```go
type ScanOptions struct {
    ChannelID int64  // Override default channel
    Limit     int    // Số message tối đa
    OffsetID  int    // Bắt đầu từ message ID này
    MinMsgID  int    // ID nhỏ nhất đã có (tránh duplicate)
}

type ChannelVideoMeta struct {
    MsgID, GrpID, ChanID, DocID, DocAccessHash
    Text, Mime, Name
    Dur, W, H, Size
    FileRef, MsgDate
    Spoiler, StreamingSupport
}
```

#### 3. Cập nhật: `backend/services/telegram/service.go`

Thêm các method:

```go
func (s *TelegramService) ScanChannel(ctx, opts) ([]ChannelVideoMeta, error)
func (s *TelegramService) ScanChannelInConnection(ctx, opts) ([]ChannelVideoMeta, error)
func (s *TelegramService) GetScanStatus() ScanStatus
func (s *TelegramService) IsScanning() bool
func (s *TelegramService) RefreshFileReference(ctx, messageID) ([]byte, error)
func (s *TelegramService) GetScanner() *ChannelScanner
func (s *TelegramService) GetAccessHash() int64
```

#### 4. Cập nhật: `backend/services/database.go`

Thêm collection mới:

```go
var TelegramChannelVideosCollection *mongo.Collection

// Indexes:
- telegram_message_id (unique)
- telegram_channel_id
- telegram_grouped_id
- is_published
- synced_at
- telegram_message_date
- caption (text search)
```

#### 5. Cập nhật: `backend/controllers/videoController.go`

Thêm các handler:

| Method                       | Handler                          | Mô tả |
| ---------------------------- | -------------------------------- | ----- |
| `SyncTelegramChannel`        | Quét channel và lưu video vào DB |
| `GetSyncStatus`              | Lấy trạng thái đồng bộ           |
| `GetTelegramVideos`          | Danh sách video đã đồng bộ       |
| `PublishTelegramVideo`       | Publish video thành video chính  |
| `StreamTelegramChannelVideo` | Stream video từ Telegram         |
| `DeleteTelegramVideo`        | Xóa video đã đồng bộ             |

#### 6. Cập nhật: `backend/routes/routes.go`

Thêm routes:

```go
// PUBLIC
videos.Get("/telegram/sync/status", videoController.GetSyncStatus)
videos.Get("/telegram/list", videoController.GetTelegramVideos)
videos.Get("/telegram/:id/stream", videoController.StreamTelegramChannelVideo)

// PROTECTED (cần đăng nhập)
videos.Post("/telegram/sync", middleware.AuthMiddleware(), videoController.SyncTelegramChannel)
videos.Post("/telegram/:id/publish", middleware.AuthMiddleware(), videoController.PublishTelegramVideo)
videos.Delete("/telegram/:id", middleware.AuthMiddleware(), videoController.DeleteTelegramVideo)
```

### Frontend

#### 1. Cập nhật: `frontend/src/services/api.js`

Thêm `telegramAPI`:

```javascript
export const telegramAPI = {
  getStatus: () => api.get("/videos/telegram/status"),
  syncChannel: (params) =>
    api.post("/videos/telegram/sync", params, { timeout: 300000 }),
  getSyncStatus: () => api.get("/videos/telegram/sync/status"),
  getVideos: (params) => api.get("/videos/telegram/list", { params }),
  publishVideo: (id, data) => api.post(`/videos/telegram/${id}/publish`, data),
  deleteVideo: (id) => api.delete(`/videos/telegram/${id}`),
  getStreamURL: (id) => `${baseURL}/videos/telegram/${id}/stream`,
};
```

#### 2. Cập nhật: `frontend/src/pages/Profile.jsx`

Thêm section "Đồng bộ Telegram" trong tab Cài đặt:

- Hiển thị trạng thái kết nối Telegram
- Nút "Đồng bộ ngay" để bắt đầu sync
- Nút "Kiểm tra kết nối"
- Hiển thị kết quả sync
- Link đến trang quản lý video đã đồng bộ

#### 3. Trang mới: `frontend/src/pages/TelegramVideos.jsx`

Trang quản lý video đã đồng bộ từ Telegram:

- Grid hiển thị video với thông tin: duration, file size, caption
- Preview video (click để xem)
- Nút Publish để chuyển thành video chính
- Nút Delete để xóa khỏi danh sách
- Modal publish với form nhập title, description, tags, genres
- Phân trang

#### 4. Cập nhật: `frontend/src/App.jsx`

Thêm route:

```jsx
<Route path="videos/telegram" element={<TelegramVideos />} />
```

#### 5. Cập nhật: `frontend/src/pages/index.js`

Export component mới:

```javascript
export { default as TelegramVideos } from "./TelegramVideos";
```

## API Endpoints

| Method | Endpoint                           | Auth | Mô tả                       |
| ------ | ---------------------------------- | ---- | --------------------------- |
| GET    | `/api/videos/telegram/status`      | No   | Trạng thái kết nối Telegram |
| POST   | `/api/videos/telegram/sync`        | Yes  | Bắt đầu đồng bộ channel     |
| GET    | `/api/videos/telegram/sync/status` | No   | Trạng thái đang đồng bộ     |
| GET    | `/api/videos/telegram/list`        | No   | Danh sách video đã sync     |
| POST   | `/api/videos/telegram/:id/publish` | Yes  | Publish video               |
| GET    | `/api/videos/telegram/:id/stream`  | No   | Stream video                |
| DELETE | `/api/videos/telegram/:id`         | Yes  | Xóa video đã sync           |

## Luồng hoạt động

### 1. Đồng bộ video

```
User click "Đồng bộ"
    → Frontend gọi POST /api/videos/telegram/sync
    → Backend: ChannelScanner.ScanChannel()
        → Telegram API: MessagesGetHistory
        → Lọc message có video
        → Extract metadata (duration, size, file_id, ...)
        → Lưu vào TelegramChannelVideosCollection
    → Trả về số video mới tìm thấy
```

### 2. Stream video

```
User click play video
    → Frontend: <video src="/api/videos/telegram/:id/stream">
    → Backend: StreamTelegramChannelVideo()
        → Lấy video info từ MongoDB
        → Parse Range header (hỗ trợ seek)
        → TelegramService.StreamVideo()
            → Telegram API: UploadGetFile (chunk by chunk)
        → Stream data về client
```

### 3. Publish video

```
User click "Publish" + nhập title
    → Frontend gọi POST /api/videos/telegram/:id/publish
    → Backend: PublishTelegramVideo()
        → Tạo Video mới từ TelegramChannelVideo
        → StorageProvider = "telegram"
        → Status = "ready"
        → Đánh dấu IsPublished = true
    → Video xuất hiện trong danh sách video chính
```

## Cấu hình cần thiết

### Environment Variables (Backend)

```env
TELEGRAM_API_ID=your_api_id
TELEGRAM_API_HASH=your_api_hash
TELEGRAM_CHANNEL_ID=-100xxxxxxxxxx
TELEGRAM_PHONE=+84xxxxxxxxx  # Hoặc TELEGRAM_BOT_TOKEN
```

### Database

Collection `telegram_channel_videos` được tạo tự động với các indexes cần thiết.

## Lưu ý

1. **File Reference**: Telegram file reference có thể hết hạn. Sử dụng `RefreshFileReference()` để làm mới.

2. **Album/Grouped Videos**: Mỗi video trong album được lưu riêng biệt với `telegram_grouped_id` chung.

3. **Streaming**: Hỗ trợ Range requests để client có thể seek video.

4. **Rate Limiting**: ChannelScanner có protection layer để tránh bị Telegram rate limit.

5. **Persistent Connection**: Backend duy trì kết nối MTProto liên tục để streaming hiệu quả.
