# Prompt Nâng Cao: Full-stack Media Website (React + Go) – Ready-to-Run

## Mục tiêu
- Xây website media quản lý **truyện tranh + video**, responsive mobile-first, UI đẹp.
- Chỉ 1 dev + 1 user.
- Full project **chạy trực tiếp**, chỉ cần `.env` → `npm install && go run main.go`.
- Tích hợp:
  - **MongoDB Atlas**: metadata truyện/video
  - **Cloudinary**: ảnh + video ngắn (<10p)
  - **Mega.nz**: video dài (>10p)
- Trang: Home, Videos, Comics, Admin/Management, Upload
- Upload:
  - Truyện: folder zip hoặc nhiều ảnh
  - Video ngắn: 3-5p, Video dài: <30p
- Backend xử lý đa luồng: upload, convert video, resize ảnh

---

## Stack
- **Frontend:** React 18 + Vite, TailwindCSS, React Router v6, React Query (TanStack)
- **Backend:** Go 1.21+, Fiber, JWT auth, bcrypt
- **Database:** MongoDB Atlas
- **Storage:** Cloudinary (ảnh/video ngắn), Mega.nz (video dài)
- **Hosting:** Local dev / Vercel (frontend), Render (backend)
- **Run commands:** `npm install` → `npm run dev` (frontend), `go run main.go` (backend)

---

## Frontend yêu cầu
1. SPA mobile-first, responsive, đẹp.
2. Trang:
   - **Home:** list truyện/video mới nhất, trending, slider video nổi bật
   - **Videos:** filter theo thể loại/độ dài, resume playback
   - **Comics:** swipe/scroll reader, bookmark page/chapter
   - **Admin:** CRUD video/truyện, thống kê lượt xem
   - **Upload:** upload folder zip truyện hoặc video, backend lưu link + metadata
3. Components reusable: Navbar, VideoCard, ComicCard, Player, ImageGallery
4. Tối ưu:
   - Lazy load ảnh/video
   - Offline cache (PWA)
   - Dark mode
   - Playlist / Collection user

---

## Backend yêu cầu
1. Go + Fiber RESTful API (hoặc GraphQL)
2. Models: User, Comic, Video
3. Controllers: userController, comicController, videoController
4. Services: cloudinaryService, megaService
5. Multi-thread xử lý upload/convert/resize với Goroutines
6. Upload workflow:
   - Truyện: zip → giải nén → upload Cloudinary → lưu link DB
   - Video ngắn <5p → Cloudinary
   - Video dài <30p → Mega.nz
7. Auth: JWT + bcrypt
8. Realtime notification upload xong (WebSocket/SSE)
9. CRUD đầy đủ + bookmark + playlist

---

## Database
- MongoDB Atlas:
  - Collections: users, comics, videos
  - Metadata: title, description, tags, images, video_link, duration, upload_date, views, bookmark_page
- Search: text search + autocomplete

---

## File Structure
/frontend
/pages
Home.jsx
Videos.jsx
Comics.jsx
Admin.jsx
Upload.jsx
/components
Navbar.jsx
VideoCard.jsx
ComicCard.jsx
Player.jsx
ImageGallery.jsx

/backend
/controllers
userController.go
comicController.go
videoController.go
/models
user.go
comic.go
video.go
/services
cloudinaryService.go
megaService.go
/routes
routes.go
main.go

---

## Environment Variables (.env)
- MONGO_URI
- JWT_SECRET
- CLOUDINARY_CLOUD_NAME
- CLOUDINARY_API_KEY
- CLOUDINARY_API_SECRET
- MEGA_EMAIL
- MEGA_PASSWORD

---

## Yêu cầu output cho AI Copilot
1. Tạo **full project chạy được ngay**:
   - Frontend + Backend, scripts start/run trực tiếp
   - Chỉ cần `.env` là chạy: `npm install && go run main.go`
2. Mobile-first, responsive, TailwindCSS + Headless UI / Flowbite
3. Backend:
   - Multi-thread upload/convert video
   - CRUD + bookmark + playlist
   - JWT auth, bcrypt
   - Stream video/truyện từ Cloudinary / Mega
   - Realtime notification upload xong
4. Frontend:
   - Lazy load, offline cache (PWA)
   - Dark mode, playlist, bookmark
   - Player resume playback
5. Comment hướng dẫn trực tiếp trong code
6. Sẵn sàng deploy cho hosting (Render, Vercel) mà không cần Docker