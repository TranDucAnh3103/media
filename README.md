# 🎬 MediaHub

<div align="center">

![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)
![React](https://img.shields.io/badge/React-18-61DAFB?logo=react&logoColor=black)
![MongoDB](https://img.shields.io/badge/MongoDB-Atlas-47A248?logo=mongodb&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-green.svg)

**Full-stack media website để quản lý Comics và Videos với tích hợp Cloud Storage**

[Demo](#demo) • [Tính năng](#-tính-năng) • [Cài đặt](#-cài-đặt) • [API Docs](#-api-documentation)

</div>

---

## 📖 Giới thiệu

**MediaHub** là một ứng dụng web full-stack cho phép người dùng quản lý, xem và chia sẻ truyện tranh (comics) và video. Ứng dụng được xây dựng với kiến trúc hiện đại, tối ưu cho hiệu suất và trải nghiệm người dùng.

### Điểm nổi bật

- 🚀 **Hiệu suất cao**: Backend Go với Goroutines xử lý đa luồng, Frontend React với lazy loading
- ☁️ **Cloud Storage thông minh**: Tự động chọn Cloudinary hoặc Mega.nz dựa trên loại nội dung
- 📱 **PWA Ready**: Có thể cài đặt như ứng dụng native, hoạt động offline
- 🎨 **UI/UX hiện đại**: Dark mode, responsive design, mobile-first

---

## ✨ Tính năng

### 🎬 Quản lý Video

- Upload video với nhiều định dạng (MP4, WebM, AVI, MKV)
- Tự động phân loại theo độ dài (short < 5p, medium 5-10p, long > 10p)
- Streaming video từ Cloudinary và Mega.nz
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

| Service           | Sử dụng                              |
| ----------------- | ------------------------------------ |
| **Cloudinary**    | Lưu trữ ảnh + video ngắn (< 10 phút) |
| **Mega.nz**       | Lưu trữ video dài (> 10 phút)        |
| **MongoDB Atlas** | Database as a Service                |

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
│   │   └── 📄 megaService.go       # Long video upload to Mega.nz
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

# Mega.nz (tùy chọn - cho video dài > 10 phút)
MEGA_EMAIL=your-mega-email@example.com
MEGA_PASSWORD=your-mega-password
```

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

### 📚 Comic Endpoints

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

### Video Upload

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

| Route                       | Component   | Mô tả                          |
| --------------------------- | ----------- | ------------------------------ |
| `/`                         | Home        | Trang chủ với trending content |
| `/videos`                   | Videos      | Danh sách video với filter     |
| `/videos/:id`               | VideoDetail | Xem video với player           |
| `/comics`                   | Comics      | Danh sách truyện               |
| `/comics/:id`               | ComicDetail | Thông tin chi tiết truyện      |
| `/comics/:id/read/:chapter` | ComicReader | Đọc chapter                    |
| `/login`                    | Login       | Đăng nhập                      |
| `/register`                 | Register    | Đăng ký                        |
| `/upload`                   | Upload      | Upload content (auth)          |
| `/dashboard`                | Dashboard   | Dashboard cá nhân (auth)       |
| `/profile`                  | Profile     | Quản lý profile (auth)         |
| `/admin`                    | Admin       | Admin panel (admin only)       |

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

### Frontend

- Lazy loading images với `OptimizedImage`
- React Query caching
- Code splitting với React.lazy
- Service Worker caching
- Optimized bundle với Vite

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

<div align="center">

Made with ❤️ using Go + React

⭐ Star this repo if you find it useful!

</div>
