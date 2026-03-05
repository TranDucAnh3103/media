# MediaHub

Full-stack media website để quản lý comics và videos với tích hợp cloud storage.

## Tech Stack

### Backend

- **Go 1.21+** với Fiber framework
- **MongoDB Atlas** - Database
- **JWT** - Authentication
- **Cloudinary** - Image & short video storage
- **Mega.nz** - Long video storage (>10min)
- **WebSocket** - Real-time notifications

### Frontend

- **React 18** với Vite
- **TailwindCSS** - Styling
- **React Router v6** - Routing
- **TanStack React Query** - Data fetching & caching
- **Zustand** - State management
- **React Player** - Video player

## Features

- 🎬 Quản lý videos (upload, stream, categories)
- 📚 Quản lý truyện tranh (chapters, đọc online)
- 🔐 Authentication (JWT, OAuth2)
- 🌙 Dark mode
- 📱 Responsive design + PWA
- 🔄 Real-time notifications
- 📑 Bookmarks & Playlists
- ⏯️ Resume playback
- 🔍 Tìm kiếm full-text
- 👤 User profiles

## Cấu trúc thư mục

```
MediaHub/
├── backend/
│   ├── main.go
│   ├── go.mod
│   ├── .env.example
│   ├── models/
│   │   ├── user.go
│   │   ├── video.go
│   │   └── comic.go
│   ├── controllers/
│   │   ├── userController.go
│   │   ├── videoController.go
│   │   └── comicController.go
│   ├── services/
│   │   ├── database.go
│   │   ├── cloudinaryService.go
│   │   └── megaService.go
│   ├── routes/
│   │   └── routes.go
│   └── middleware/
│       └── auth.go
│
├── frontend/
│   ├── package.json
│   ├── vite.config.js
│   ├── tailwind.config.js
│   ├── postcss.config.js
│   ├── index.html
│   └── src/
│       ├── main.jsx
│       ├── App.jsx
│       ├── index.css
│       ├── services/
│       │   └── api.js
│       ├── store/
│       │   ├── authStore.js
│       │   ├── themeStore.js
│       │   └── playerStore.js
│       ├── components/
│       │   ├── Layout.jsx
│       │   ├── Navbar.jsx
│       │   ├── VideoCard.jsx
│       │   ├── ComicCard.jsx
│       │   ├── Player.jsx
│       │   ├── ImageGallery.jsx
│       │   └── ProtectedRoute.jsx
│       └── pages/
│           ├── Home.jsx
│           ├── Videos.jsx
│           ├── VideoDetail.jsx
│           ├── Comics.jsx
│           ├── ComicDetail.jsx
│           ├── ComicReader.jsx
│           ├── Upload.jsx
│           ├── Admin.jsx
│           ├── Login.jsx
│           ├── Register.jsx
│           ├── Profile.jsx
│           └── NotFound.jsx
│
└── README.md
```

## Cài đặt

### Backend

1. Cài đặt Go 1.21+
2. Copy `.env.example` thành `.env` và cấu hình:
   ```bash
   cd backend
   cp .env.example .env
   ```
3. Cập nhật các biến môi trường trong `.env`
4. Chạy server:
   ```bash
   go mod download
   go run main.go
   ```

### Frontend

1. Cài đặt Node.js 18+
2. Install dependencies:
   ```bash
   cd frontend
   npm install
   ```
3. Chạy development server:
   ```bash
   npm run dev
   ```
4. Build production:
   ```bash
   npm run build
   ```

## API Endpoints

### Authentication

- `POST /api/auth/register` - Đăng ký
- `POST /api/auth/login` - Đăng nhập
- `GET /api/auth/me` - Lấy thông tin user hiện tại

### Videos

- `GET /api/videos` - Danh sách videos
- `GET /api/videos/:id` - Chi tiết video
- `POST /api/videos` - Upload video (auth required)
- `PUT /api/videos/:id` - Cập nhật video
- `DELETE /api/videos/:id` - Xóa video
- `POST /api/videos/:id/like` - Like video
- `POST /api/videos/:id/comment` - Comment
- `GET /api/videos/:id/upload-progress` - Tracking upload progress

### Comics

- `GET /api/comics` - Danh sách truyện
- `GET /api/comics/:id` - Chi tiết truyện
- `POST /api/comics` - Tạo truyện mới
- `PUT /api/comics/:id` - Cập nhật truyện
- `DELETE /api/comics/:id` - Xóa truyện
- `POST /api/comics/:id/chapters` - Upload chapter
- `GET /api/comics/:id/chapters/:num` - Lấy chapter

### User

- `GET /api/user/profile` - Profile với bookmarks, playlists
- `PUT /api/user/profile` - Cập nhật profile
- `POST /api/user/bookmarks` - Thêm bookmark
- `DELETE /api/user/bookmarks/:id` - Xóa bookmark
- `POST /api/user/playlists` - Tạo playlist
- `POST /api/user/history` - Cập nhật lịch sử xem

### Admin

- `GET /api/admin/stats` - Thống kê
- `GET /api/admin/users` - Danh sách users
- `PUT /api/admin/users/:id/ban` - Ban/Unban user
- `GET /api/admin/videos` - Quản lý videos
- `GET /api/admin/comics` - Quản lý truyện

## Environment Variables

### Backend (.env)

```
PORT=3000
MONGODB_URI=mongodb+srv://username:password@cluster.mongodb.net/mediahub
JWT_SECRET=your-jwt-secret-key
CLOUDINARY_CLOUD_NAME=your-cloud-name
CLOUDINARY_API_KEY=your-api-key
CLOUDINARY_API_SECRET=your-api-secret
MEGA_EMAIL=your-mega-email
MEGA_PASSWORD=your-mega-password
```

### Frontend

API URL được cấu hình trong `vite.config.js` với proxy đến backend.

## License

MIT
