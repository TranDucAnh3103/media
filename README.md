# 🎬 MediaHub

<div align="center">

![Version](https://img.shields.io/badge/version-2.0.0-blue.svg)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)
![React](https://img.shields.io/badge/React-18-61DAFB?logo=react&logoColor=black)
![MongoDB](https://img.shields.io/badge/MongoDB-Atlas-47A248?logo=mongodb&logoColor=white)
![Telegram](https://img.shields.io/badge/Telegram-MTProto-26A5E4?logo=telegram&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-green.svg)

**Full-stack media platform với Telegram MTProto storage và HTTP Range streaming tối ưu**

[Demo](#demo) • [Tính năng](#-tính-năng) • [Cài đặt](#-cài-đặt) • [API Docs](#-api-documentation)

</div>

---

## 📖 Giới thiệu

**MediaHub** là một ứng dụng web full-stack cho phép người dùng quản lý, xem và chia sẻ truyện tranh (comics) và video. Ứng dụng được xây dựng với kiến trúc hiện đại, tối ưu cho hiệu suất và trải nghiệm người dùng.

### 🆕 What's New in v2.0

- 📡 **Telegram MTProto Integration**: Upload và stream video trực tiếp từ Telegram channel
- ⚡ **HTTP Range Streaming**: Hỗ trợ seek/skip video với range requests
- 🔄 **Smart Sync**: Đồng bộ video từ Telegram channel với duplicate detection
- 🛡️ **Rate Limiting & Flood Protection**: Bảo vệ khỏi Telegram API limits
- 💾 **Unlimited Storage**: Sử dụng Telegram làm cloud storage miễn phí không giới hạn

### Điểm nổi bật

- 🚀 **Hiệu suất cao**: Backend Go với Goroutines xử lý đa luồng, Frontend React với lazy loading
- 📡 **Telegram Storage**: Upload video lên Telegram channel, stream qua MTProto API
- ☁️ **Multi-Cloud Support**: Telegram (primary), Cloudinary (images), Mega.nz (legacy)
- 📱 **PWA Ready**: Có thể cài đặt như ứng dụng native, hoạt động offline
- 🎨 **UI/UX hiện đại**: Dark mode, responsive design, mobile-first

---

## ✨ Tính năng

### 🎬 Quản lý Video

- Upload video với nhiều định dạng (MP4, WebM, AVI, MKV)
- **📡 Telegram Storage**: Lưu trữ video trên Telegram channel (không giới hạn dung lượng)
- **⚡ HTTP Range Streaming**: Hỗ trợ seek/skip với byte-range requests
- **🔄 Channel Sync**: Đồng bộ video từ Telegram channel có sẵn
- Tự động phân loại theo độ dài (short < 5p, medium 5-10p, long > 10p)
- Resume playback - tiếp tục xem từ vị trí đã dừng
- Like, comment, view count tracking

### 📚 Quản lý Truyện Tranh

- Upload truyện dạng folder ZIP hoặc nhiều ảnh
- Đọc truyện online với swipe/scroll navigation
- Bookmark trang/chapter đang đọc
- Hệ thống rating và đánh giá
- Quản lý chapters riêng biệt

### 👤 Người dùng

- Đăng ký/Đăng nhập với JWT authentication
- User profile với avatar upload
- Bookmarks cho video và truyện
- Playlist cá nhân
- Lịch sử xem (watch history)
- Phân quyền Admin/User

### 🎯 Tính năng khác

- 🔍 Full-text search với autocomplete
- 🌙 Dark/Light mode
- 📱 Responsive design cho mọi thiết bị
- 🔄 Real-time notifications qua WebSocket
- 📊 Admin dashboard với thống kê

---

## 🛠 Tech Stack

### Backend

| Technology        | Mô tả                                 |
| ----------------- | ------------------------------------- |
| **Go 1.21+**      | Ngôn ngữ lập trình hiệu suất cao      |
| **Fiber v2**      | Web framework siêu nhanh cho Go       |
| **MongoDB Atlas** | Cloud database NoSQL                  |
| **JWT**           | JSON Web Tokens cho authentication    |
| **WebSocket**     | Real-time bidirectional communication |
| **bcrypt**        | Password hashing an toàn              |

### Frontend

| Technology            | Mô tả                                       |
| --------------------- | ------------------------------------------- |
| **React 18**          | UI library với hooks và concurrent features |
| **Vite 5**            | Build tool siêu nhanh                       |
| **TailwindCSS 3**     | Utility-first CSS framework                 |
| **React Router v6**   | Client-side routing                         |
| **TanStack Query v5** | Data fetching và caching                    |
| **Zustand**           | Lightweight state management                |
| **React Player**      | Video player component                      |

### Cloud Services

| Service           | Sử dụng                                           |
| ----------------- | ------------------------------------------------- |
| **Telegram**      | 📡 Primary video storage (unlimited, MTProto API) |
| **Cloudinary**    | Lưu trữ ảnh, thumbnails, video ngắn               |
| **Mega.nz**       | Legacy storage (backward compatibility)           |
| **MongoDB Atlas** | Database as a Service                             |

### Telegram Integration

| Component            | Mô tả                                     |
| -------------------- | ----------------------------------------- |
| **gotd/td**          | MTProto client library cho Go             |
| **Channel Scanner**  | Quét và đồng bộ video từ Telegram channel |
| **HTTP Streamer**    | Stream video với Range request support    |
| **Rate Limiter**     | Token bucket algorithm cho API protection |
| **Flood Protection** | Backoff strategy khi gặp FloodWait errors |

---

## 📁 Cấu trúc dự án

```
MediaHub/
├── 📄 README.md                    # Documentation
├── 📄 provider.md                  # Prompt & requirements
│
├── 📂 backend/                     # Go Backend
│   ├── 📄 main.go                  # Entry point, Fiber app setup
│   ├── 📄 go.mod                   # Go modules
│   ├── 📄 seed.go                  # Database seeding
│   │
│   ├── 📂 cmd/
│   │   └── 📂 seed/
│   │       └── 📄 main.go          # Seed command
│   │
│   ├── 📂 controllers/             # Request handlers
│   │   ├── 📄 userController.go    # Auth, profile, bookmarks
│   │   ├── 📄 videoController.go   # Video CRUD, upload, stream
│   │   └── 📄 comicController.go   # Comic CRUD, chapters
│   │
│   ├── 📂 models/                  # Data models
│   │   ├── 📄 user.go              # User, Bookmark, Playlist, WatchHistory
│   │   ├── 📄 video.go             # Video, Comment, UploadProgress
│   │   └── 📄 comic.go             # Comic, Chapter, ComicImage
│   │
│   ├── 📂 services/                # Business logic
│   │   ├── 📄 database.go          # MongoDB connection & indexes
│   │   ├── 📄 cloudinaryService.go # Image/video upload to Cloudinary
│   │   ├── 📄 megaService.go       # Long video upload to Mega.nz
│   │   └── 📂 telegram/            # 📡 Telegram MTProto integration
│   │       ├── 📄 service.go       # Main Telegram service
│   │       ├── 📄 client.go        # MTProto client management
│   │       ├── 📄 uploader.go      # Video upload to channel
│   │       ├── 📄 streamer.go      # HTTP Range streaming
│   │       ├── 📄 scanner.go       # Channel video scanner
│   │       ├── 📄 ratelimit.go     # Token bucket rate limiter
│   │       ├── 📄 floodwait.go     # FloodWait error handling
│   │       ├── 📄 cache.go         # File reference cache
│   │       └── 📄 types.go         # Shared types & structs
│   │
│   ├── 📂 routes/
│   │   └── 📄 routes.go            # API route definitions
│   │
│   └── 📂 middleware/
│       └── 📄 auth.go              # JWT authentication middleware
│
└── 📂 frontend/                    # React Frontend
    ├── 📄 index.html               # HTML entry point
    ├── 📄 package.json             # npm dependencies
    ├── 📄 vite.config.js           # Vite configuration
    ├── 📄 tailwind.config.js       # TailwindCSS config
    ├── 📄 postcss.config.js        # PostCSS config
    │
    ├── 📂 public/
    │   ├── 📄 manifest.json        # PWA manifest
    │   └── 📄 sw.js                # Service Worker
    │
    └── 📂 src/
        ├── 📄 main.jsx             # React entry point
        ├── 📄 App.jsx              # Main app component với routes
        ├── 📄 index.css            # Global styles + Tailwind
        │
        ├── 📂 components/          # Reusable components
        │   ├── 📄 Layout.jsx       # Main layout wrapper
        │   ├── 📄 Navbar.jsx       # Navigation bar
        │   ├── 📄 BottomNav.jsx    # Mobile bottom navigation
        │   ├── 📄 MobileHeader.jsx # Mobile header
        │   ├── 📄 VideoCard.jsx    # Video thumbnail card
        │   ├── 📄 ComicCard.jsx    # Comic thumbnail card
        │   ├── 📄 Player.jsx       # Video player wrapper
        │   ├── 📄 ImageGallery.jsx # Comic image viewer
        │   ├── 📄 OptimizedImage.jsx # Lazy loading image
        │   └── 📄 ProtectedRoute.jsx # Auth route guard
        │
        ├── 📂 pages/               # Page components
        │   ├── 📄 Home.jsx         # Landing page
        │   ├── 📄 Videos.jsx       # Video listing
        │   ├── 📄 VideoDetail.jsx  # Single video player
        │   ├── 📄 Comics.jsx       # Comic listing
        │   ├── 📄 ComicDetail.jsx  # Comic info page
        │   ├── 📄 ComicReader.jsx  # Comic chapter reader
        │   ├── 📄 Upload.jsx       # Upload video/comic
        │   ├── 📄 Dashboard.jsx    # User dashboard
        │   ├── 📄 Admin.jsx        # Admin panel
        │   ├── 📄 Login.jsx        # Login page
        │   ├── 📄 Register.jsx     # Registration page
        │   ├── 📄 Profile.jsx      # User profile
        │   └── 📄 NotFound.jsx     # 404 page
        │
        ├── 📂 services/
        │   └── 📄 api.js           # Axios API client
        │
        ├── 📂 store/               # Zustand stores
        │   ├── 📄 authStore.js     # Authentication state
        │   ├── 📄 themeStore.js    # Theme preferences
        │   └── 📄 playerStore.js   # Video player state
        │
        └── 📂 hooks/
            └── 📄 useImageCache.js # Image caching hook
```

---

## 🚀 Cài đặt

### Yêu cầu hệ thống

- **Go** 1.21 trở lên
- **Node.js** 18 trở lên
- **MongoDB Atlas** account (hoặc MongoDB local)
- **Cloudinary** account
- **Mega.nz** account (tùy chọn, cho video dài)

### 1. Clone repository

```bash
git clone https://github.com/your-username/mediahub.git
cd mediahub
```

### 2. Cấu hình Backend

```bash
# Di chuyển vào thư mục backend
cd backend

# Tạo file .env từ template
cp .env.example .env

# Chỉnh sửa file .env với thông tin của bạn
```

**Nội dung file `.env`:**

```env
# Server
PORT=8080

# MongoDB Atlas
MONGO_URI=mongodb+srv://<username>:<password>@<cluster>.mongodb.net/?retryWrites=true&w=majority
MONGO_DB_NAME=media_db

# JWT Secret (tạo random string dài)
JWT_SECRET=your-super-secret-jwt-key-here

# Cloudinary
CLOUDINARY_CLOUD_NAME=your-cloud-name
CLOUDINARY_API_KEY=123456789012345
CLOUDINARY_API_SECRET=your-cloudinary-api-secret

# Mega.nz (legacy - backward compatibility)
MEGA_EMAIL=your-mega-email@example.com
MEGA_PASSWORD=your-mega-password

# Telegram MTProto (v2.0 - PRIMARY STORAGE)
TELEGRAM_API_ID=12345678
TELEGRAM_API_HASH=your-telegram-api-hash
TELEGRAM_PHONE=+84xxxxxxxxx
TELEGRAM_CHANNEL_ID=-1001234567890
TELEGRAM_SESSION_FILE=telegram_session/session.json
```

> 💡 **Lấy Telegram API credentials**: Truy cập https://my.telegram.org → API Development tools

```bash
# Cài đặt dependencies và chạy server
go mod download
go run main.go
```

Server sẽ chạy tại `http://localhost:8080`

### 3. Cấu hình Frontend

```bash
# Di chuyển vào thư mục frontend
cd frontend

# Cài đặt dependencies
npm install

# Chạy development server
npm run dev
```

Frontend sẽ chạy tại `http://localhost:5173`

### 4. Seed dữ liệu mẫu (tùy chọn)

```bash
cd backend
go run -tags seed cmd/seed/main.go
```

---

## 📡 API Documentation

### Base URL

```
Development: http://localhost:8080/api
Production:  https://your-domain.com/api
```

### Authentication

Sử dụng Bearer Token trong header:

```
Authorization: Bearer <your-jwt-token>
```

---

### 🔐 Auth Endpoints

| Method | Endpoint         | Mô tả                 | Auth |
| ------ | ---------------- | --------------------- | ---- |
| `POST` | `/auth/register` | Đăng ký tài khoản mới | ❌   |
| `POST` | `/auth/login`    | Đăng nhập             | ❌   |

<details>
<summary><b>POST /auth/register</b></summary>

**Request Body:**

```json
{
  "username": "johndoe",
  "email": "john@example.com",
  "password": "password123"
}
```

**Response (201):**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "65abc123def456...",
    "username": "johndoe",
    "email": "john@example.com",
    "role": "user",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

</details>

<details>
<summary><b>POST /auth/login</b></summary>

**Request Body:**

```json
{
  "email": "john@example.com",
  "password": "password123"
}
```

**Response (200):**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "65abc123def456...",
    "username": "johndoe",
    "email": "john@example.com",
    "role": "user"
  }
}
```

</details>

---

### 👤 User Endpoints

| Method   | Endpoint                             | Mô tả                   | Auth |
| -------- | ------------------------------------ | ----------------------- | ---- |
| `GET`    | `/user/profile`                      | Lấy thông tin profile   | ✅   |
| `PUT`    | `/user/profile`                      | Cập nhật profile        | ✅   |
| `POST`   | `/user/bookmarks`                    | Thêm bookmark           | ✅   |
| `DELETE` | `/user/bookmarks/:contentId`         | Xóa bookmark            | ✅   |
| `POST`   | `/user/playlists`                    | Tạo playlist mới        | ✅   |
| `POST`   | `/user/playlists/:playlistId/videos` | Thêm video vào playlist | ✅   |

---

### 🎬 Video Endpoints

| Method   | Endpoint                      | Mô tả                       | Auth |
| -------- | ----------------------------- | --------------------------- | ---- |
| `GET`    | `/videos`                     | Danh sách video (có filter) | ❌   |
| `GET`    | `/videos/trending`            | Video trending              | ❌   |
| `GET`    | `/videos/latest`              | Video mới nhất              | ❌   |
| `GET`    | `/videos/my`                  | Video của tôi               | ✅   |
| `GET`    | `/videos/:id`                 | Chi tiết video              | ❌   |
| `POST`   | `/videos/upload`              | Upload video mới            | ✅   |
| `PUT`    | `/videos/:id`                 | Cập nhật video              | ✅   |
| `DELETE` | `/videos/:id`                 | Xóa video                   | ✅   |
| `POST`   | `/videos/:id/like`            | Like video                  | ✅   |
| `POST`   | `/videos/:id/comments`        | Thêm comment                | ✅   |
| `GET`    | `/videos/upload/progress/:id` | Tiến trình upload           | ❌   |
| `GET`    | `/videos/stream/mega/:hash`   | Stream video từ Mega        | ❌   |

<details>
<summary><b>GET /videos - Query Parameters</b></summary>

| Param           | Type     | Mô tả                           |
| --------------- | -------- | ------------------------------- |
| `search`        | string   | Tìm kiếm theo title             |
| `genres`        | string[] | Filter theo genres              |
| `duration_type` | string   | `short`, `medium`, `long`       |
| `quality`       | string   | `360p`, `480p`, `720p`, `1080p` |
| `sort_by`       | string   | `views`, `likes`, `created_at`  |
| `order`         | string   | `asc`, `desc`                   |
| `page`          | int      | Trang (mặc định: 1)             |
| `limit`         | int      | Số lượng/trang (mặc định: 10)   |

**Ví dụ:**

```
GET /api/videos?genres=action,comedy&duration_type=short&sort_by=views&order=desc&page=1&limit=20
```

</details>

<details>
<summary><b>POST /videos/upload</b></summary>

**Request:** `multipart/form-data`

| Field         | Type   | Mô tả                            |
| ------------- | ------ | -------------------------------- |
| `video`       | File   | File video (MP4, WebM, AVI, MKV) |
| `thumbnail`   | File   | Ảnh thumbnail (tùy chọn)         |
| `title`       | string | Tiêu đề video                    |
| `description` | string | Mô tả                            |
| `tags`        | string | Tags phân cách bằng dấu phẩy     |
| `genres`      | string | Genres phân cách bằng dấu phẩy   |

**Response (201):**

```json
{
  "message": "Video upload started",
  "upload_id": "upload_abc123",
  "video_id": "65def789..."
}
```

</details>

---

### � Telegram Endpoints (v2.0)

| Method   | Endpoint                       | Mô tả                             | Auth |
| -------- | ------------------------------ | --------------------------------- | ---- |
| `POST`   | `/telegram/sync`               | Đồng bộ video từ Telegram channel | ✅   |
| `GET`    | `/telegram/sync/status`        | Trạng thái đồng bộ                | ✅   |
| `GET`    | `/telegram/videos`             | Danh sách video đã sync           | ✅   |
| `POST`   | `/telegram/videos/:id/publish` | Publish video ra hệ thống chính   | ✅   |
| `GET`    | `/telegram/videos/:id/stream`  | Stream video với Range support    | ❌   |
| `DELETE` | `/telegram/videos/:id`         | Xóa video khỏi danh sách sync     | ✅   |

<details>
<summary><b>POST /telegram/sync - Đồng bộ channel</b></summary>

**Response (200):**

```json
{
  "success": true,
  "message": "Synced 5 new videos from Telegram channel",
  "new_videos_count": 5,
  "skipped_count": 45,
  "total_scanned": 100,
  "sync_duration": "2.5s"
}
```

</details>

<details>
<summary><b>GET /telegram/videos/:id/stream - HTTP Range Streaming</b></summary>

**Headers hỗ trợ:**

```
Range: bytes=0-1048575        # Request first 1MB
Range: bytes=1048576-         # Request from 1MB to end
```

**Response Headers:**

```
HTTP/1.1 206 Partial Content
Accept-Ranges: bytes
Content-Range: bytes 0-1048575/52428800
Content-Length: 1048576
Content-Type: video/mp4
```

**Tính năng:**

- ✅ Seek/skip tới bất kỳ vị trí nào trong video
- ✅ Resume download khi mất kết nối
- ✅ Buffering thông minh
- ✅ Chunk size tối ưu (512KB - 2MB)

</details>

---

### �📚 Comic Endpoints

| Method   | Endpoint                           | Mô tả                 | Auth |
| -------- | ---------------------------------- | --------------------- | ---- |
| `GET`    | `/comics`                          | Danh sách truyện      | ❌   |
| `GET`    | `/comics/trending`                 | Truyện trending       | ❌   |
| `GET`    | `/comics/latest`                   | Truyện mới cập nhật   | ❌   |
| `GET`    | `/comics/my`                       | Truyện của tôi        | ✅   |
| `GET`    | `/comics/:id`                      | Chi tiết truyện       | ❌   |
| `GET`    | `/comics/:id/chapters/:chapterNum` | Đọc chapter           | ❌   |
| `POST`   | `/comics`                          | Tạo truyện mới        | ✅   |
| `POST`   | `/comics/upload`                   | Upload truyện với ảnh | ✅   |
| `POST`   | `/comics/:id/chapters`             | Thêm chapter mới      | ✅   |
| `PUT`    | `/comics/:id`                      | Cập nhật truyện       | ✅   |
| `DELETE` | `/comics/:id`                      | Xóa truyện            | ✅   |
| `POST`   | `/comics/:id/like`                 | Like truyện           | ✅   |

---

### 👑 Admin Endpoints

| Method | Endpoint        | Mô tả              | Auth     |
| ------ | --------------- | ------------------ | -------- |
| `GET`  | `/admin/stats`  | Thống kê tổng quan | ✅ Admin |
| `GET`  | `/admin/users`  | Danh sách users    | ✅ Admin |
| `GET`  | `/admin/videos` | Quản lý videos     | ✅ Admin |
| `GET`  | `/admin/comics` | Quản lý truyện     | ✅ Admin |

<details>
<summary><b>GET /admin/stats Response</b></summary>

```json
{
  "users": 150,
  "comics": 45,
  "videos": 230
}
```

</details>

---

### 🔌 WebSocket

**Endpoint:** `ws://localhost:8080/ws`

Sử dụng cho real-time notifications về tiến trình upload và các sự kiện khác.

**Message Format:**

```json
{
  "type": "upload_progress",
  "data": {
    "upload_id": "abc123",
    "progress": 75,
    "status": "processing"
  }
}
```

---

## 💾 Database Schema

### Users Collection

```javascript
{
  _id: ObjectId,
  username: String,          // Unique, 3-50 chars
  email: String,             // Unique, valid email
  password: String,          // bcrypt hash
  avatar: String,            // Cloudinary URL
  role: String,              // "admin" | "user"
  liked_videos: [ObjectId],
  liked_comics: [ObjectId],
  bookmarks: [{
    content_id: ObjectId,
    content_type: String,    // "comic" | "video"
    page: Number,
    chapter: Number,
    timestamp: Number        // Seconds for video
  }],
  playlists: [{
    _id: ObjectId,
    name: String,
    video_ids: [ObjectId]
  }],
  watch_history: [{
    content_id: ObjectId,
    content_type: String,
    watched_at: Date,
    progress: Number         // Percentage
  }],
  created_at: Date,
  updated_at: Date
}
```

### Videos Collection

```javascript
{
  _id: ObjectId,
  title: String,
  description: String,
  thumbnail: String,         // Cloudinary URL
  video_url: String,         // Cloudinary or Mega URL
  cloudinary_public_id: String,
  mega_hash: String,
  storage_type: String,      // "cloudinary" | "mega"
  duration: Number,          // Seconds
  duration_type: String,     // "short" | "medium" | "long"
  quality: String,           // "360p" | "480p" | "720p" | "1080p"
  file_size: Number,         // Bytes
  tags: [String],
  genres: [String],
  views: Number,
  likes: Number,
  dislikes: Number,
  comments: [{
    _id: ObjectId,
    user_id: ObjectId,
    username: String,
    content: String,
    likes: Number,
    created_at: Date
  }],
  status: String,            // "processing" | "ready" | "error"
  uploaded_by: ObjectId,
  created_at: Date,
  updated_at: Date
}
```

### Telegram Channel Videos Collection (v2.0)

```javascript
{
  _id: ObjectId,
  telegram_message_id: Number,    // Unique message ID trong channel
  telegram_grouped_id: Number,    // Group ID nếu thuộc album
  telegram_channel_id: Number,    // Channel ID nguồn
  telegram_file_id: String,       // File ID để download
  telegram_file_ref: String,      // File reference (có thời hạn)
  telegram_access_hash: Number,   // Access hash cho file
  caption: String,                // Caption từ message
  duration: Number,               // Độ dài video (seconds)
  file_size: Number,              // Kích thước file (bytes)
  width: Number,                  // Video width
  height: Number,                 // Video height
  mime_type: String,              // "video/mp4"
  file_name: String,              // Tên file gốc
  telegram_message_date: Date,    // Ngày upload lên Telegram
  synced_at: Date,                // Ngày sync vào hệ thống
  has_spoiler: Boolean,           // Có spoiler warning không
  supports_streaming: Boolean,    // Hỗ trợ streaming không
  is_published: Boolean,          // Đã publish ra Videos collection chưa
  published_video_id: ObjectId    // Link tới video đã publish
}
```

**Indexes:**

- `telegram_message_id` (unique) - Tránh duplicate khi sync
- `synced_at` (descending) - Sort theo thời gian sync
- `is_published` - Filter video chưa publish

### Comics Collection

```javascript
{
  _id: ObjectId,
  title: String,
  description: String,
  author: String,
  cover_image: String,       // Cloudinary URL
  tags: [String],
  genres: [String],
  status: String,            // "ongoing" | "completed"
  chapters: [{
    _id: ObjectId,
    number: Number,
    title: String,
    images: [{
      page: Number,
      url: String,           // Cloudinary URL
      public_id: String,
      width: Number,
      height: Number
    }],
    views: Number,
    uploaded_at: Date
  }],
  views: Number,
  likes: Number,
  rating: Number,            // 0-5
  rating_count: Number,
  uploaded_by: ObjectId,
  created_at: Date,
  updated_at: Date
}
```

---

## 🔄 Upload Workflow

### Video Upload (v2.0 - Telegram)

```
┌──────────────┐         ┌──────────────┐         ┌──────────────┐
│   Frontend   │         │   Backend    │         │   Telegram   │
└──────┬───────┘         └──────┬───────┘         └──────┬───────┘
       │                        │                        │
       │  POST /videos/upload   │                        │
       │ ─────────────────────► │                        │
       │                        │                        │
       │     upload_id          │                        │
       │ ◄───────────────────── │                        │
       │                        │                        │
       │                        │  MTProto Upload        │
       │                        │  (chunked, resumable)  │
       │                        │ ─────────────────────► │
       │                        │                        │
       │                        │  Progress callback     │
       │                        │ ◄───────────────────── │
       │                        │                        │
       │  WebSocket: progress   │                        │
       │ ◄───────────────────── │                        │
       │                        │                        │
       │                        │  File ID + metadata    │
       │                        │ ◄───────────────────── │
       │                        │                        │
       │  WebSocket: completed  │                        │
       │ ◄───────────────────── │                        │
       │                        │                        │
```

### Video Streaming (v2.0 - HTTP Range)

```
┌──────────────┐         ┌──────────────┐         ┌──────────────┐
│    Client    │         │   Backend    │         │   Telegram   │
│  (Browser)   │         │  (Streamer)  │         │   (MTProto)  │
└──────┬───────┘         └──────┬───────┘         └──────┬───────┘
       │                        │                        │
       │  GET /stream/:id       │                        │
       │  Range: bytes=0-       │                        │
       │ ─────────────────────► │                        │
       │                        │                        │
       │                        │  Calculate offset      │
       │                        │  & chunk size          │
       │                        │                        │
       │                        │  messages.getDocument  │
       │                        │  (precise_seek=true)   │
       │                        │ ─────────────────────► │
       │                        │                        │
       │                        │  Partial file data     │
       │                        │ ◄───────────────────── │
       │                        │                        │
       │  206 Partial Content   │                        │
       │  Content-Range: bytes  │                        │
       │ ◄───────────────────── │                        │
       │                        │                        │
       │  [User seeks to 50%]   │                        │
       │  Range: bytes=26214400-│                        │
       │ ─────────────────────► │                        │
       │                        │                        │
       │                        │  Seek to offset        │
       │                        │ ─────────────────────► │
       │                        │                        │
       │  206 Partial Content   │                        │
       │ ◄───────────────────── │                        │
       │                        │                        │
```

### Telegram Channel Sync (v2.0)

```
┌──────────────┐         ┌──────────────┐         ┌──────────────┐
│   Frontend   │         │   Backend    │         │   Telegram   │
└──────┬───────┘         └──────┬───────┘         └──────┬───────┘
       │                        │                        │
       │  POST /telegram/sync   │                        │
       │ ─────────────────────► │                        │
       │                        │                        │
       │                        │  Get last synced ID    │
       │                        │  from MongoDB          │
       │                        │                        │
       │                        │  messages.getHistory   │
       │                        │  (min_id = last_synced)│
       │                        │ ─────────────────────► │
       │                        │                        │
       │                        │  New messages only     │
       │                        │ ◄───────────────────── │
       │                        │                        │
       │                        │  Extract video msgs    │
       │                        │  Check duplicates      │
       │                        │  InsertOne new only    │
       │                        │                        │
       │  {new: 5, skipped: 0}  │                        │
       │ ◄───────────────────── │                        │
       │                        │                        │
```

### Video Upload (Legacy - Cloudinary/Mega)

```
┌──────────────┐         ┌──────────────┐         ┌──────────────┐
│   Frontend   │         │   Backend    │         │    Storage   │
└──────┬───────┘         └──────┬───────┘         └──────┬───────┘
       │                        │                        │
       │  POST /videos/upload   │                        │
       │ ─────────────────────► │                        │
       │                        │                        │
       │     upload_id          │                        │
       │ ◄───────────────────── │                        │
       │                        │                        │
       │                        │  Check duration        │
       │                        │ ─────────────────────► │
       │                        │                        │
       │                        │      if < 10min        │
       │                        │ ─────► Cloudinary      │
       │                        │                        │
       │                        │      if > 10min        │
       │                        │ ─────► Mega.nz         │
       │                        │                        │
       │  WebSocket: progress   │                        │
       │ ◄───────────────────── │                        │
       │                        │                        │
       │  WebSocket: completed  │                        │
       │ ◄───────────────────── │                        │
       │                        │                        │
```

### Comic Upload

```
┌──────────────┐         ┌──────────────┐         ┌──────────────┐
│   Frontend   │         │   Backend    │         │  Cloudinary  │
└──────┬───────┘         └──────┬───────┘         └──────┬───────┘
       │                        │                        │
       │  POST /comics/upload   │                        │
       │  (multipart/form-data) │                        │
       │ ─────────────────────► │                        │
       │                        │                        │
       │                        │  Parallel upload       │
       │                        │  (5 concurrent)        │
       │                        │ ─────────────────────► │
       │                        │                        │
       │                        │  Transform & optimize  │
       │                        │ ◄───────────────────── │
       │                        │                        │
       │     Comic created      │                        │
       │ ◄───────────────────── │                        │
       │                        │                        │
```

---

## 🎨 Frontend Pages

| Route                       | Component      | Mô tả                               |
| --------------------------- | -------------- | ----------------------------------- |
| `/`                         | Home           | Trang chủ với trending content      |
| `/videos`                   | Videos         | Danh sách video với filter          |
| `/videos/:id`               | VideoDetail    | Xem video với player                |
| `/comics`                   | Comics         | Danh sách truyện                    |
| `/comics/:id`               | ComicDetail    | Thông tin chi tiết truyện           |
| `/comics/:id/read/:chapter` | ComicReader    | Đọc chapter                         |
| `/login`                    | Login          | Đăng nhập                           |
| `/register`                 | Register       | Đăng ký                             |
| `/upload`                   | Upload         | Upload content (auth)               |
| `/dashboard`                | Dashboard      | Dashboard cá nhân (auth)            |
| `/profile`                  | Profile        | Quản lý profile + Telegram sync     |
| `/telegram-videos`          | TelegramVideos | 📡 Quản lý video từ Telegram (v2.0) |
| `/admin`                    | Admin          | Admin panel (admin only)            |

---

## 🌙 Dark Mode

Ứng dụng hỗ trợ Dark/Light mode với:

- Tự động detect system preference
- Lưu preference vào localStorage
- Smooth transition animation
- Consistent color scheme với TailwindCSS

```javascript
// Toggle dark mode
const { darkMode, toggleDarkMode } = useThemeStore();
```

---

## 📱 PWA Features

MediaHub là Progressive Web App với:

- ✅ Installable trên mobile/desktop
- ✅ Offline support với Service Worker
- ✅ App-like experience
- ✅ Push notifications ready
- ✅ Splash screen

**Manifest:** [frontend/public/manifest.json](frontend/public/manifest.json)

---

## 🔒 Security

- **Password hashing**: bcrypt với cost factor 10
- **JWT tokens**: Expiry 24h, HMAC-SHA256
- **CORS**: Cấu hình origins cụ thể
- **Input validation**: Server-side validation
- **File type validation**: Chỉ cho phép file types cụ thể
- **Rate limiting**: Có thể thêm middleware

### Telegram Security (v2.0)

- **Session encryption**: MTProto session được mã hóa lưu local
- **API Rate Limiting**: Token bucket với flood protection
- **File Reference Refresh**: Tự động refresh khi hết hạn
- **Channel Access**: Chỉ đọc từ channel được cấu hình
- **Safe Mode**: Tự động giảm tải khi gặp lỗi liên tục

---

## 🚢 Deployment

### Backend (Render/Railway/VPS)

```bash
# Build binary
go build -o main .

# Run with environment variables
PORT=8080 ./main
```

### Frontend (Vercel/Netlify)

```bash
# Build
npm run build

# Output: dist/
```

**Vercel config (`vercel.json`):**

```json
{
  "rewrites": [{ "source": "/(.*)", "destination": "/" }]
}
```

---

## 🧪 Development

### Backend Hot Reload

```bash
# Install air
go install github.com/cosmtrek/air@latest

# Run with hot reload
air
```

### Frontend

```bash
npm run dev     # Development với HMR
npm run lint    # ESLint check
npm run build   # Production build
npm run preview # Preview production build
```

---

## 📊 Performance Optimizations

### Backend

- Connection pooling MongoDB (50 connections)
- Goroutines cho parallel upload
- Semaphore giới hạn concurrent operations
- Efficient file streaming

### Telegram Streaming (v2.0)

| Optimization                | Mô tả                                             |
| --------------------------- | ------------------------------------------------- |
| **Adaptive Chunk Size**     | Tự động điều chỉnh 512KB - 2MB dựa trên bandwidth |
| **Request Coalescing**      | Gộp nhiều requests nhỏ thành một                  |
| **File Reference Cache**    | Cache file refs với LRU eviction                  |
| **Token Bucket Rate Limit** | Giới hạn 20 req/s, burst 30                       |
| **Exponential Backoff**     | Xử lý FloodWait errors tự động                    |
| **Connection Pooling**      | Reuse MTProto connections                         |
| **Precise Seek**            | Byte-level seeking với offset alignment           |

### API Rate Limiting (v2.0)

```
┌─────────────────────────────────────────────────────────┐
│                 Token Bucket Algorithm                  │
├─────────────────────────────────────────────────────────┤
│  Rate: 20 tokens/second                                 │
│  Burst: 30 tokens (buffer cho spike)                    │
│  FloodWait: Automatic backoff + retry                   │
│  Safe Mode: Giảm 50% rate khi gặp lỗi liên tục         │
└─────────────────────────────────────────────────────────┘
```

### Frontend

- Lazy loading images với `OptimizedImage`
- React Query caching
- Code splitting với React.lazy
- Service Worker caching
- Optimized bundle với Vite
- **Video preload** với range requests (v2.0)
- **Adaptive buffering** dựa trên network speed (v2.0)

---

## 🤝 Contributing

1. Fork repository
2. Tạo branch mới: `git checkout -b feature/amazing-feature`
3. Commit changes: `git commit -m 'Add amazing feature'`
4. Push to branch: `git push origin feature/amazing-feature`
5. Tạo Pull Request

---

## 📝 License

MIT License - xem file [LICENSE](LICENSE) để biết thêm chi tiết.

---

## 👨‍💻 Author

**MediaHub Team**

---

## 📚 Tài liệu bổ sung

- 📖 [TELEGRAM_MIGRATION_README.md](TELEGRAM_MIGRATION_README.md) - Chi tiết về Telegram integration
- 📖 [syncTele.md](syncTele.md) - Hướng dẫn sync video từ Telegram channel

---

<div align="center">

**MediaHub v2.0** - Made with ❤️ using Go + React + Telegram MTProto

⭐ Star this repo if you find it useful!

</div>
