Đã xử lý xong các lỗi và cảnh báo trong backend. Tóm tắt thay đổi:

1. floodwait.go – Copy mutex
Tạo floodWaitStatsInternal với mutex riêng, tách khỏi FloodWaitStats (DTO)
FloodWaitStats chỉ chứa dữ liệu (không có mutex) nên an toàn khi copy/return
GetStats() và Stats() trả về DTO, không còn copy mutex
2. ratelimit.go – Copy mutex
Tạo rateLimitMetricsInternal với mutex riêng
RateLimitMetrics chỉ còn dữ liệu
GetMetrics() và Stats() trả về DTO
3. main.go – Race condition
Thêm clientsMu sync.RWMutex để bảo vệ map clients
Mọi truy cập clients đều qua lock
Goroutine broadcast sao chép danh sách client trước khi gửi để tránh deadlock
4. ctx.Locals("userID") – Tránh panic
Thêm middleware.GetUserID(ctx) (type assertion an toàn)
Thêm middleware.RequireUserID(ctx) (trả 401 nếu thiếu user)
Cập nhật toàn bộ controllers: videoController, userController, comicController
5. comicController.go – Biến đã khai báo
Đổi _, err := thành _, err = ở InsertOne vì err đã được khai báo từ RequireUserID
Kết quả: go build ./... và go vet ./... đều pass, không còn lỗi hoặc cảnh báo.

Lưu ý: Theo error.md, vẫn còn vài điểm có thể cải thiện sau:

comicController.go – Upload zip chưa implement (TODO)
Goroutine/DB – Một số goroutine gọi DB/HTTP nhưng chưa log lỗi