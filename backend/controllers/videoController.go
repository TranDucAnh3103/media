package controllers

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"media-backend/middleware"
	"media-backend/models"
	"media-backend/services"
	"media-backend/services/telegram"

	"github.com/gofiber/fiber/v2"
	"github.com/gotd/td/tg"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// telegramSyncState - Shared state for async sync (thread-safe)
type telegramSyncState struct {
	mu              sync.RWMutex
	IsRunning       bool
	SyncID          string
	CurrentProgress int
	Total           int
	NewCount        int
	LastError       string
	StartedAt       time.Time
}

func (s *telegramSyncState) get() (running bool, id string, progress, total, newCount int, lastErr string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.IsRunning, s.SyncID, s.CurrentProgress, s.Total, s.NewCount, s.LastError
}

func (s *telegramSyncState) setRunning(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.IsRunning = true
	s.SyncID = id
	s.CurrentProgress = 0
	s.Total = 0
	s.NewCount = 0
	s.LastError = ""
	s.StartedAt = time.Now()
}

func (s *telegramSyncState) setProgress(progress, total, newCount int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CurrentProgress = progress
	s.Total = total
	s.NewCount = newCount
}

func (s *telegramSyncState) setDone(lastErr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.IsRunning = false
	if lastErr != "" {
		s.LastError = lastErr
	}
}

// VideoController - Controller xử lý video operations
type VideoController struct {
	cloudinary      *services.CloudinaryService
	mega            *services.MegaService     // Legacy - giữ lại cho backward compatibility
	telegram        *telegram.TelegramService // New - Telegram storage
	uploading       sync.Map                  // Track upload progress
	telegramSync    telegramSyncState         // Async sync state
}

// NewVideoController - Tạo instance mới
func NewVideoController() *VideoController {
	cloudinary, _ := services.NewCloudinaryService()
	mega, _ := services.NewMegaService() // Legacy

	// Initialize Telegram service
	tgSvc, err := telegram.NewTelegramService()
	if err != nil {
		fmt.Printf("Warning: Failed to initialize Telegram service: %v\n", err)
	}

	return &VideoController{
		cloudinary: cloudinary,
		mega:       mega,
		telegram:   tgSvc,
	}
}

// InitTelegramConnection - Khởi động persistent connection cho Telegram streaming
// Gọi method này khi server start
func (c *VideoController) InitTelegramConnection(ctx context.Context) error {
	if c.telegram == nil {
		return fmt.Errorf("telegram service not initialized")
	}

	fmt.Println("[VideoController] Starting Telegram persistent connection for streaming...")
	err := c.telegram.StartPersistentConnection(ctx)
	if err != nil {
		return fmt.Errorf("failed to start Telegram connection: %w", err)
	}
	fmt.Println("[VideoController] Telegram connection ready for streaming")
	return nil
}

// StopTelegramConnection - Dừng Telegram connection khi server shutdown
func (c *VideoController) StopTelegramConnection() {
	if c.telegram != nil {
		c.telegram.StopPersistentConnection()
	}
}

// GetVideos - Lấy danh sách video với filter & pagination
// GET /api/videos
func (c *VideoController) GetVideos(ctx *fiber.Ctx) error {
	// Parse query params
	page := ctx.QueryInt("page", 1)
	limit := ctx.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100
	}
	skip := (page - 1) * limit

	// Build filter
	filter := bson.M{"status": "ready"} // Chỉ lấy video đã sẵn sàng

	if genres := ctx.Query("genres"); genres != "" {
		filter["genres"] = bson.M{"$in": strings.Split(genres, ",")}
	}
	if durationType := ctx.Query("duration_type"); durationType != "" {
		filter["duration_type"] = durationType
	}
	if quality := ctx.Query("quality"); quality != "" {
		filter["quality"] = quality
	}
	if search := ctx.Query("search"); search != "" {
		filter["$text"] = bson.M{"$search": search}
	}

	// Build sort
	sortField := ctx.Query("sort_by", "created_at")
	sortOrder := -1
	if ctx.Query("order") == "asc" {
		sortOrder = 1
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Count total
	total, err := services.VideosCollection.CountDocuments(dbCtx, filter)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to count videos",
		})
	}

	// Find videos
	findOptions := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: sortField, Value: sortOrder}})

	cursor, err := services.VideosCollection.Find(dbCtx, filter, findOptions)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch videos",
		})
	}
	defer cursor.Close(dbCtx)

	var videos []models.Video
	if err := cursor.All(dbCtx, &videos); err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to decode videos",
		})
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return ctx.JSON(models.VideoListResponse{
		Videos:     videos,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	})
}

// GetVideo - Lấy chi tiết một video
// GET /api/videos/:id
func (c *VideoController) GetVideo(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid video ID",
		})
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var video models.Video
	err = services.VideosCollection.FindOne(dbCtx, bson.M{"_id": objID}).Decode(&video)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{
			"error": "Video not found",
		})
	}

	// Tăng views (background)
	go func() {
		updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		services.VideosCollection.UpdateOne(updateCtx, bson.M{"_id": objID}, bson.M{
			"$inc": bson.M{"views": 1},
		})
	}()

	return ctx.JSON(video)
}

// StreamMegaVideo - Stream video từ Mega (Legacy)
// GET /api/videos/stream/mega/:hash
func (c *VideoController) StreamMegaVideo(ctx *fiber.Ctx) error {
	hash := ctx.Params("hash")
	if hash == "" {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Video hash is required",
		})
	}

	if c.mega == nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Mega service not available",
		})
	}

	// Get file size for Content-Length header
	size, err := c.mega.GetFileSize(hash)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{
			"error": "Video not found: " + err.Error(),
		})
	}

	// Set headers for video streaming
	ctx.Set("Content-Type", "video/mp4")
	ctx.Set("Content-Length", fmt.Sprintf("%d", size))
	ctx.Set("Accept-Ranges", "bytes")
	ctx.Set("Cache-Control", "public, max-age=3600")

	// Stream video content
	ctx.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		err := c.mega.DownloadToWriter(hash, w)
		if err != nil {
			// Log error but can't send response at this point
			fmt.Printf("Error streaming Mega video: %v\n", err)
		}
		w.Flush()
	})

	return nil
}

// StreamVideo - Stream video với hỗ trợ đa provider (Telegram, Cloudinary, Mega)
// GET /api/videos/stream/:id
func (c *VideoController) StreamVideo(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid video ID",
		})
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var video models.Video
	err = services.VideosCollection.FindOne(dbCtx, bson.M{"_id": objID}).Decode(&video)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{
			"error": "Video not found",
		})
	}

	// Route to appropriate streaming method based on storage provider
	switch video.StorageProvider {
	case models.StorageProviderTelegram:
		return c.streamFromTelegram(ctx, &video)
	case models.StorageProviderMega:
		if video.MegaHash != "" {
			return c.streamFromMega(ctx, video.MegaHash)
		}
		return ctx.Redirect(video.VideoURL)
	case models.StorageProviderCloudinary:
		return ctx.Redirect(video.VideoURL)
	default:
		// Legacy: check storage_type and mega_hash
		if video.StorageType == "mega" && video.MegaHash != "" {
			return c.streamFromMega(ctx, video.MegaHash)
		}
		if video.VideoURL != "" {
			return ctx.Redirect(video.VideoURL)
		}
		return ctx.Status(404).JSON(fiber.Map{
			"error": "Video stream not available",
		})
	}
}

// GetVideoThumbnail - Lấy thumbnail; nếu chưa có (video Telegram) thì tạo từ frame đầu rồi redirect
// GET /api/videos/:id/thumbnail
func (c *VideoController) GetVideoThumbnail(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"error": "Invalid video ID"})
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var video models.Video
	err = services.VideosCollection.FindOne(dbCtx, bson.M{"_id": objID}).Decode(&video)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{"error": "Video not found"})
	}

	// Đã có thumbnail → redirect ngay
	if video.Thumbnail != "" {
		return ctx.Redirect(video.Thumbnail, 302)
	}

	// Chỉ tạo thumbnail cho video Telegram
	if video.StorageProvider != models.StorageProviderTelegram {
		return ctx.Status(404).JSON(fiber.Map{"error": "Thumbnail not available"})
	}

	if c.telegram == nil || c.cloudinary == nil {
		return ctx.Status(500).JSON(fiber.Map{"error": "Service not available for thumbnail generation"})
	}

	if !c.telegram.IsConnected() {
		if startErr := c.telegram.StartPersistentConnection(context.Background()); startErr != nil {
			return ctx.Status(503).JSON(fiber.Map{"error": "Telegram not connected"})
		}
		if !c.telegram.IsConnected() {
			return ctx.Status(503).JSON(fiber.Map{"error": "Telegram connection failed"})
		}
	}

	// Tạo thumbnail: tải chunk đầu → extract frame → upload Cloudinary → cập nhật DB
	thumbnailURL, genErr := c.generateTelegramVideoThumbnail(ctx.UserContext(), &video)
	if genErr != nil {
		fmt.Printf("[Thumbnail] Failed to generate for video %s: %v\n", id, genErr)
		return ctx.Status(500).JSON(fiber.Map{"error": "Failed to generate thumbnail: " + genErr.Error()})
	}

	return ctx.Redirect(thumbnailURL, 302)
}

// generateTelegramVideoThumbnail - Tải chunk đầu từ Telegram, extract frame, upload Cloudinary, cập nhật DB
func (c *VideoController) generateTelegramVideoThumbnail(ctx context.Context, video *models.Video) (string, error) {
	const chunkSize = 2 * 1024 * 1024 // 2MB đủ cho FFmpeg đọc frame đầu
	fileSize := video.FileSize
	if fileSize == 0 {
		size, err := c.telegram.GetFileSize(ctx, video.TelegramMessageID)
		if err != nil {
			return "", fmt.Errorf("get file size: %w", err)
		}
		fileSize = size
	}

	toRead := chunkSize
	if fileSize < int64(toRead) {
		toRead = int(fileSize)
	}
	if toRead <= 0 {
		return "", fmt.Errorf("video file size is 0")
	}

	tempDir := os.TempDir()
	tempPath := filepath.Join(tempDir, fmt.Sprintf("tg_video_%s.mp4", video.ID.Hex()))
	defer os.Remove(tempPath)

	f, err := os.Create(tempPath)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}

	err = c.telegram.StreamVideo(ctx, telegram.StreamRequest{
		ChannelID: video.TelegramChannelID,
		MessageID: video.TelegramMessageID,
		FileID:    video.TelegramFileID,
		Start:     0,
		End:       int64(toRead) - 1,
	}, f)
	f.Close()
	if err != nil {
		return "", fmt.Errorf("stream chunk: %w", err)
	}

	thumbPath, err := telegram.ExtractThumbnail(tempPath, tempDir)
	if err != nil {
		return "", fmt.Errorf("extract thumbnail: %w", err)
	}
	defer os.Remove(thumbPath)

	thumbResult, err := c.cloudinary.UploadImageFromPath(ctx, thumbPath, "video_thumbnails")
	if err != nil {
		return "", fmt.Errorf("upload to Cloudinary: %w", err)
	}
	if thumbResult.SecureURL == "" {
		return "", fmt.Errorf("Cloudinary returned empty URL")
	}

	updateCtx, updateCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer updateCancel()

	_, err = services.VideosCollection.UpdateOne(
		updateCtx,
		bson.M{"_id": video.ID},
		bson.M{"$set": bson.M{
			"thumbnail":   thumbResult.SecureURL,
			"updated_at":  time.Now(),
		}},
	)
	if err != nil {
		return thumbResult.SecureURL, nil // vẫn redirect, nhưng DB chưa update
	}

	return thumbResult.SecureURL, nil
}

// GetTelegramStatus - Kiểm tra trạng thái Telegram service
// GET /api/videos/telegram/status
func (c *VideoController) GetTelegramStatus(ctx *fiber.Ctx) error {
	status := fiber.Map{
		"telegram_initialized": c.telegram != nil,
		"telegram_connected":   false,
		"message":              "Telegram service not initialized",
	}

	if c.telegram != nil {
		connected := c.telegram.IsConnected()
		status["telegram_connected"] = connected
		if connected {
			status["message"] = "Telegram service is ready for streaming"
		} else {
			status["message"] = "Telegram service initialized but not connected. Waiting for connection..."
		}
	}

	return ctx.JSON(status)
}

// streamFromTelegram - Stream video từ Telegram với Range request support
func (c *VideoController) streamFromTelegram(ctx *fiber.Ctx, video *models.Video) error {
	// Debug logging
	fmt.Printf("[StreamTelegram] Starting stream for video %s (Message ID: %d)\n", video.ID.Hex(), video.TelegramMessageID)

	if c.telegram == nil {
		fmt.Println("[StreamTelegram] ERROR: Telegram service is nil")
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Telegram service not initialized",
		})
	}

	if !c.telegram.IsConnected() {
		fmt.Println("[StreamTelegram] ERROR: Telegram service not connected")
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Telegram service not connected. Please try again later.",
		})
	}

	// Get file size
	fileSize := video.FileSize
	if fileSize == 0 {
		fmt.Printf("[StreamTelegram] FileSize is 0, fetching from Telegram for message %d\n", video.TelegramMessageID)
		size, err := c.telegram.GetFileSize(ctx.UserContext(), video.TelegramMessageID)
		if err != nil {
			fmt.Printf("[StreamTelegram] ERROR: Failed to get file size: %v\n", err)
			return ctx.Status(500).JSON(fiber.Map{
				"error": "Failed to get file size: " + err.Error(),
			})
		}
		fileSize = size
		fmt.Printf("[StreamTelegram] Got file size: %d bytes\n", fileSize)
	}

	// Parse Range header
	rangeHeader := ctx.Get("Range")
	start, end, err := telegram.ParseRangeHeader(rangeHeader, fileSize)
	if err != nil {
		fmt.Printf("[StreamTelegram] ERROR: Invalid range header: %s\n", rangeHeader)
		return ctx.Status(416).JSON(fiber.Map{
			"error": "Range not satisfiable",
		})
	}
	fmt.Printf("[StreamTelegram] Range: %d-%d (total: %d)\n", start, end, fileSize)

	// Determine content type
	contentType := video.MimeType
	if contentType == "" {
		contentType = "video/mp4"
	}

	// Set response headers
	ctx.Set("Content-Type", contentType)
	ctx.Set("Accept-Ranges", "bytes")
	ctx.Set("Cache-Control", "public, max-age=3600")

	if rangeHeader != "" {
		// Partial content response
		ctx.Set("Content-Length", fmt.Sprintf("%d", end-start+1))
		ctx.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
		ctx.Status(206) // Partial Content
	} else {
		// Full content response
		ctx.Set("Content-Length", fmt.Sprintf("%d", fileSize))
		ctx.Status(200)
	}

	// Stream video content
	fmt.Printf("[StreamTelegram] Starting stream from Telegram (Channel: %d, Message: %d)\n",
		video.TelegramChannelID, video.TelegramMessageID)

	streamCtx := ctx.UserContext()
	ctx.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		err := c.telegram.StreamVideo(streamCtx, telegram.StreamRequest{
			ChannelID: video.TelegramChannelID,
			MessageID: video.TelegramMessageID,
			FileID:    video.TelegramFileID,
			Start:     start,
			End:       end,
		}, w)
		if err != nil {
			fmt.Printf("[StreamTelegram] ERROR: Streaming failed: %v\n", err)
		} else {
			fmt.Printf("[StreamTelegram] Stream completed successfully\n")
		}
		w.Flush()
	})

	return nil
}

// streamFromMega - Stream video từ Mega (legacy helper)
func (c *VideoController) streamFromMega(ctx *fiber.Ctx, hash string) error {
	if c.mega == nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Mega service not available",
		})
	}

	// Get file size
	size, err := c.mega.GetFileSize(hash)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{
			"error": "Video not found: " + err.Error(),
		})
	}

	// Set headers
	ctx.Set("Content-Type", "video/mp4")
	ctx.Set("Content-Length", fmt.Sprintf("%d", size))
	ctx.Set("Accept-Ranges", "bytes")
	ctx.Set("Cache-Control", "public, max-age=3600")

	// Stream video content
	ctx.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		err := c.mega.DownloadToWriter(hash, w)
		if err != nil {
			fmt.Printf("Error streaming Mega video: %v\n", err)
		}
		w.Flush()
	})

	return nil
}

// GetMyVideos - Lấy danh sách video của user hiện tại
// GET /api/videos/my
// Đối với app cá nhân: trả về TẤT CẢ video có status "ready"
func (c *VideoController) GetMyVideos(ctx *fiber.Ctx) error {
	// Xác thực user (không filter theo user vì đây là app cá nhân)
	_, _ = middleware.GetUserID(ctx)

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Lấy tất cả video có status "ready" (app cá nhân - không cần phân quyền)
	filter := bson.M{"status": "ready"}
	cursor, err := services.VideosCollection.Find(dbCtx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch videos",
		})
	}
	defer cursor.Close(dbCtx)

	var videos []models.Video
	if err := cursor.All(dbCtx, &videos); err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to decode videos",
		})
	}

	return ctx.JSON(fiber.Map{
		"videos": videos,
	})
}

// UploadVideo - Upload video mới (multi-thread processing)
// POST /api/videos/upload
func (c *VideoController) UploadVideo(ctx *fiber.Ctx) error {
	userID, err := middleware.RequireUserID(ctx)
	if err != nil {
		return err
	}
	userObjID, _ := primitive.ObjectIDFromHex(userID)

	// Parse form data
	title := ctx.FormValue("title")
	description := ctx.FormValue("description")
	tags := strings.Split(ctx.FormValue("tags"), ",")
	genres := strings.Split(ctx.FormValue("genres"), ",")

	if title == "" {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Title is required",
		})
	}

	// Get uploaded file
	file, err := ctx.FormFile("video")
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Video file is required",
		})
	}

	// Validate file type
	ext := strings.ToLower(filepath.Ext(file.Filename))
	validExts := map[string]bool{
		".mp4": true, ".webm": true, ".mov": true, ".avi": true, ".mkv": true,
	}
	if !validExts[ext] {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid video format. Supported: mp4, webm, mov, avi, mkv",
		})
	}

	// CRITICAL: Save file to temp BEFORE starting goroutine
	// (multipart file data is lost after HTTP response)
	fmt.Printf("[Upload] Saving uploaded file to temp: %s (%d bytes)\n", file.Filename, file.Size)

	tempDir := os.TempDir()
	tempPath := filepath.Join(tempDir, fmt.Sprintf("upload_%d_%s", time.Now().UnixNano(), file.Filename))

	src, err := file.Open()
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to open uploaded file",
		})
	}

	tempFile, err := os.Create(tempPath)
	if err != nil {
		src.Close()
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to create temp file",
		})
	}

	written, err := tempFile.ReadFrom(src)
	tempFile.Close()
	src.Close()

	if err != nil || written == 0 {
		os.Remove(tempPath)
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to save uploaded file",
		})
	}

	fmt.Printf("[Upload] Saved %d bytes to temp file: %s\n", written, tempPath)

	// Create video record with processing status
	now := time.Now()
	videoID := primitive.NewObjectID()
	video := models.Video{
		ID:          videoID,
		Title:       title,
		Description: description,
		Tags:        tags,
		Genres:      genres,
		Status:      "processing",
		UploadedBy:  userObjID,
		FileSize:    file.Size,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = services.VideosCollection.InsertOne(dbCtx, video)
	if err != nil {
		os.Remove(tempPath)
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to create video record",
		})
	}

	// Upload video in background (multi-thread)
	// Pass temp file path instead of multipart.FileHeader
	go c.processVideoUpload(videoID, tempPath, file.Filename, file.Size, userID)

	return ctx.Status(202).JSON(fiber.Map{
		"message":  "Video upload started",
		"video_id": videoID.Hex(),
		"status":   "processing",
	})
}

// processVideoUpload - Xử lý upload video trong background (Goroutine)
// tempPath: đường dẫn file tạm đã được lưu trước khi vào goroutine
// fileName: tên file gốc
// fileSize: kích thước file
func (c *VideoController) processVideoUpload(videoID primitive.ObjectID, tempPath string, fileName string, fileSize int64, userID string) {
	fmt.Printf("[Upload] Starting background upload for video %s\n", videoID.Hex())
	fmt.Printf("[Upload] Temp file: %s, Size: %d bytes\n", tempPath, fileSize)

	// Ensure temp file is cleaned up at the end
	defer func() {
		os.Remove(tempPath)
		fmt.Printf("[Upload] Cleaned up temp file: %s\n", tempPath)
	}()

	// Track progress
	progressID := videoID.Hex()
	c.uploading.Store(progressID, models.UploadProgress{
		ID:       progressID,
		Status:   "uploading",
		Progress: 0,
		Message:  "Starting upload...",
	})

	defer func() {
		// Clean up progress tracking after some time
		time.Sleep(5 * time.Minute)
		c.uploading.Delete(progressID)
	}()

	// Verify temp file exists
	if _, err := os.Stat(tempPath); os.IsNotExist(err) {
		fmt.Printf("[Upload] ERROR: Temp file not found: %s\n", tempPath)
		c.updateVideoStatus(videoID, "error", "Temp file not found")
		return
	}

	// Determine duration type based on file size
	// (rough estimate: 1MB = ~10 seconds for compressed video)
	estimatedDuration := int(fileSize / (1024 * 1024) * 10) // seconds
	durationType := "short"
	if estimatedDuration > 300 {
		durationType = "medium"
	}
	if estimatedDuration > 600 {
		durationType = "long"
	}

	var videoURL, thumbnail, storageProvider string
	var duration int
	var telegramMessageID int
	var telegramFileID string
	var telegramChannelID int64
	var mimeType string
	var width, height int

	// Debug log storage availability
	fmt.Printf("[Upload] Checking Telegram connection...\n")
	fmt.Printf("[Upload] Telegram service: %v, Connected: %v\n", c.telegram != nil, c.telegram != nil && c.telegram.IsConnected())

	// ============ UPLOAD TO TELEGRAM ONLY ============
	if c.telegram == nil || !c.telegram.IsConnected() {
		fmt.Println("[Upload] ERROR: Telegram service not connected!")
		fmt.Println("[Upload] Make sure TELEGRAM_* environment variables are configured")
		c.updateVideoStatus(videoID, "error", "Telegram không được kết nối. Kiểm tra cấu hình server.")
		return
	}

	fmt.Printf("[Upload] Using Telegram for upload\n")
	c.uploading.Store(progressID, models.UploadProgress{
		ID:       progressID,
		Status:   "uploading",
		Progress: 10,
		Message:  "Preparing to upload to Telegram...",
	})

	// Progress callback
	progressCb := func(p telegram.UploadProgress) {
		c.uploading.Store(progressID, models.UploadProgress{
			ID:       progressID,
			Status:   "uploading",
			Progress: 10 + p.Percent*0.8, // 10-90%
			Message:  fmt.Sprintf("Uploading to Telegram: %.1f%%", p.Percent),
		})
	}

	// Upload to Telegram
	result, err := c.telegram.UploadVideo(context.Background(), telegram.VideoUploadRequest{
		FilePath:   tempPath,
		FileName:   fileName,
		Caption:    "", // Optional caption
		ChannelID:  c.telegram.GetChannelID(),
		ProgressCb: progressCb,
	})

	if err != nil {
		fmt.Printf("[Upload] ERROR: Telegram upload failed: %v\n", err)
		c.updateVideoStatus(videoID, "error", fmt.Sprintf("Telegram upload failed: %v", err))
		return
	}

	fmt.Printf("[Upload] Telegram upload successful! MessageID: %d, FileID: %s\n", result.MessageID, result.FileID)

	storageProvider = models.StorageProviderTelegram
	telegramMessageID = result.MessageID
	telegramFileID = result.FileID
	telegramChannelID = c.telegram.GetChannelID()
	mimeType = result.MimeType
	duration = result.Duration
	width = result.Width
	height = result.Height
	videoURL = fmt.Sprintf("/api/videos/stream/%s", videoID.Hex())

	// Extract metadata using ffprobe if available
	c.uploading.Store(progressID, models.UploadProgress{
		ID:       progressID,
		Status:   "processing",
		Progress: 92,
		Message:  "Extracting metadata...",
	})

	if telegram.IsFFprobeAvailable() {
		metadata, metaErr := telegram.ExtractVideoMetadata(tempPath)
		if metaErr == nil && metadata != nil {
			if metadata.Duration > 0 {
				duration = metadata.Duration
			}
			if metadata.Width > 0 {
				width = metadata.Width
			}
			if metadata.Height > 0 {
				height = metadata.Height
			}
		}
	}

	// Generate and upload thumbnail to Cloudinary
	c.uploading.Store(progressID, models.UploadProgress{
		ID:       progressID,
		Status:   "processing",
		Progress: 95,
		Message:  "Generating thumbnail...",
	})

	if telegram.IsFFmpegAvailable() && c.cloudinary != nil {
		thumbnailPath, thumbErr := telegram.ExtractThumbnail(tempPath, os.TempDir())
		if thumbErr == nil {
			thumbResult, uploadErr := c.cloudinary.UploadImageFromPath(context.Background(), thumbnailPath, "video_thumbnails")
			if uploadErr == nil && thumbResult.SecureURL != "" {
				thumbnail = thumbResult.SecureURL
				fmt.Printf("[Upload] Thumbnail generated: %s\n", thumbnail)
			} else {
				fmt.Printf("[Upload] Failed to upload thumbnail: %v\n", uploadErr)
			}
			os.Remove(thumbnailPath) // Clean up thumbnail temp file
		} else {
			fmt.Printf("[Upload] Failed to extract thumbnail: %v\n", thumbErr)
		}
	} else {
		fmt.Println("[Upload] FFmpeg not available, skipping thumbnail")
	}

	// Temp file cleanup is handled by defer at the start

	// Update duration type based on actual duration
	if duration <= 300 {
		durationType = "short"
	} else if duration <= 600 {
		durationType = "medium"
	} else {
		durationType = "long"
	}

	// Update video record
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"video_url":           videoURL,
			"thumbnail":           thumbnail,
			"storage_type":        storageProvider, // Legacy field
			"storage_provider":    storageProvider,
			"telegram_channel_id": telegramChannelID,
			"telegram_message_id": telegramMessageID,
			"telegram_file_id":    telegramFileID,
			"mime_type":           mimeType,
			"duration":            duration,
			"duration_type":       durationType,
			"width":               width,
			"height":              height,
			"status":              "ready",
			"updated_at":          time.Now(),
		},
	}

	_, err = services.VideosCollection.UpdateOne(ctx, bson.M{"_id": videoID}, update)
	if err != nil {
		c.updateVideoStatus(videoID, "error", "Failed to update video record")
		return
	}

	c.uploading.Store(progressID, models.UploadProgress{
		ID:       progressID,
		Status:   "completed",
		Progress: 100,
		Message:  "Upload completed",
		VideoID:  videoID.Hex(),
	})

	// TODO: Send WebSocket notification
}

// updateVideoStatus - Cập nhật status video
func (c *VideoController) updateVideoStatus(videoID primitive.ObjectID, status string, message string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	services.VideosCollection.UpdateOne(ctx, bson.M{"_id": videoID}, bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	})

	c.uploading.Store(videoID.Hex(), models.UploadProgress{
		ID:      videoID.Hex(),
		Status:  status,
		Message: message,
	})
}

// GetUploadProgress - Lấy tiến trình upload
// GET /api/videos/upload/progress/:id
func (c *VideoController) GetUploadProgress(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	progress, ok := c.uploading.Load(id)
	if !ok {
		// Check database for status
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return ctx.Status(400).JSON(fiber.Map{
				"error": "Invalid ID",
			})
		}

		dbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var video models.Video
		err = services.VideosCollection.FindOne(dbCtx, bson.M{"_id": objID}).Decode(&video)
		if err != nil {
			return ctx.Status(404).JSON(fiber.Map{
				"error": "Video not found",
			})
		}

		return ctx.JSON(models.UploadProgress{
			ID:      id,
			Status:  video.Status,
			Message: "Check video status",
			VideoID: id,
		})
	}

	return ctx.JSON(progress)
}

// UpdateVideo - Cập nhật thông tin video
// PUT /api/videos/:id
func (c *VideoController) UpdateVideo(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid video ID",
		})
	}

	var req models.VideoUpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	update := bson.M{
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	if req.Title != "" {
		update["$set"].(bson.M)["title"] = req.Title
	}
	if req.Description != "" {
		update["$set"].(bson.M)["description"] = req.Description
	}
	if len(req.Tags) > 0 {
		update["$set"].(bson.M)["tags"] = req.Tags
	}
	if len(req.Genres) > 0 {
		update["$set"].(bson.M)["genres"] = req.Genres
	}
	if req.Thumbnail != "" {
		update["$set"].(bson.M)["thumbnail"] = req.Thumbnail
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := services.VideosCollection.UpdateOne(dbCtx, bson.M{"_id": objID}, update)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to update video",
		})
	}

	if result.MatchedCount == 0 {
		return ctx.Status(404).JSON(fiber.Map{
			"error": "Video not found",
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Video updated successfully",
	})
}

// DeleteVideo - Xóa video
// DELETE /api/videos/:id
func (c *VideoController) DeleteVideo(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid video ID",
		})
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get video to delete from storage
	var video models.Video
	err = services.VideosCollection.FindOne(dbCtx, bson.M{"_id": objID}).Decode(&video)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{
			"error": "Video not found",
		})
	}

	// Delete from storage (background)
	go func() {
		deleteCtx := context.Background()

		switch video.StorageProvider {
		case models.StorageProviderTelegram:
			// Delete from Telegram
			if c.telegram != nil && video.TelegramMessageID != 0 {
				err := c.telegram.DeleteVideo(deleteCtx, video.TelegramMessageID)
				if err != nil {
					fmt.Printf("Error deleting video from Telegram: %v\n", err)
				}
			}
		case models.StorageProviderCloudinary:
			// Delete from Cloudinary
			if c.cloudinary != nil && video.CloudinaryPublicID != "" {
				c.cloudinary.DeleteResource(deleteCtx, video.CloudinaryPublicID, "video")
			}
		case models.StorageProviderMega:
			// Delete from Mega
			if c.mega != nil && video.MegaHash != "" {
				err := c.mega.DeleteFile(video.MegaHash)
				if err != nil {
					fmt.Printf("Error deleting video from Mega: %v\n", err)
				}
			}
		default:
			// Legacy: check storage_type
			if video.StorageType == "cloudinary" && c.cloudinary != nil && video.CloudinaryPublicID != "" {
				c.cloudinary.DeleteResource(deleteCtx, video.CloudinaryPublicID, "video")
			} else if video.StorageType == "mega" && c.mega != nil && video.MegaHash != "" {
				c.mega.DeleteFile(video.MegaHash)
			}
		}

		// Also delete thumbnail from Cloudinary if exists
		if video.Thumbnail != "" && c.cloudinary != nil {
			// Extract public ID from thumbnail URL if needed
			// c.cloudinary.DeleteResource(deleteCtx, thumbnailPublicID, "image")
		}
	}()

	// Delete from database
	result, err := services.VideosCollection.DeleteOne(dbCtx, bson.M{"_id": objID})
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to delete video",
		})
	}

	if result.DeletedCount == 0 {
		return ctx.Status(404).JSON(fiber.Map{
			"error": "Video not found",
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Video deleted successfully",
	})
}

// LikeVideo - Like/Unlike video (toggle)
// POST /api/videos/:id/like
func (c *VideoController) LikeVideo(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	userID, err := middleware.RequireUserID(ctx)
	if err != nil {
		return err
	}

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid video ID",
		})
	}

	userObjID, _ := primitive.ObjectIDFromHex(userID)

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if user already liked this video
	var user models.User
	err = services.UsersCollection.FindOne(dbCtx, bson.M{"_id": userObjID}).Decode(&user)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Check if video is already liked
	alreadyLiked := false
	for _, likedID := range user.LikedVideos {
		if likedID == objID {
			alreadyLiked = true
			break
		}
	}

	if alreadyLiked {
		// Unlike: remove from user's liked list and decrement count
		_, err = services.UsersCollection.UpdateOne(dbCtx, bson.M{"_id": userObjID}, bson.M{
			"$pull": bson.M{"liked_videos": objID},
		})
		if err != nil {
			return ctx.Status(500).JSON(fiber.Map{
				"error": "Failed to unlike video",
			})
		}

		_, err = services.VideosCollection.UpdateOne(dbCtx, bson.M{"_id": objID}, bson.M{
			"$inc": bson.M{"likes": -1},
		})
		if err != nil {
			return ctx.Status(500).JSON(fiber.Map{
				"error": "Failed to update video likes",
			})
		}

		return ctx.JSON(fiber.Map{
			"message": "Video unliked",
			"liked":   false,
		})
	}

	// Like: add to user's liked list and increment count
	_, err = services.UsersCollection.UpdateOne(dbCtx, bson.M{"_id": userObjID}, bson.M{
		"$addToSet": bson.M{"liked_videos": objID},
	})
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to like video",
		})
	}

	_, err = services.VideosCollection.UpdateOne(dbCtx, bson.M{"_id": objID}, bson.M{
		"$inc": bson.M{"likes": 1},
	})
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to update video likes",
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Video liked",
		"liked":   true,
	})
}

// AddComment - Thêm comment
// POST /api/videos/:id/comments
func (c *VideoController) AddComment(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	userID, err := middleware.RequireUserID(ctx)
	if err != nil {
		return err
	}

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid video ID",
		})
	}

	userObjID, _ := primitive.ObjectIDFromHex(userID)

	var req struct {
		Content string `json:"content"`
	}
	if err := ctx.BodyParser(&req); err != nil || req.Content == "" {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Content is required",
		})
	}

	// Get username
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var user models.User
	services.UsersCollection.FindOne(dbCtx, bson.M{"_id": userObjID}).Decode(&user)

	comment := models.Comment{
		ID:        primitive.NewObjectID(),
		UserID:    userObjID,
		Username:  user.Username,
		Content:   req.Content,
		Likes:     0,
		CreatedAt: time.Now(),
	}

	_, err = services.VideosCollection.UpdateOne(dbCtx, bson.M{"_id": objID}, bson.M{
		"$push": bson.M{"comments": comment},
	})
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to add comment",
		})
	}

	return ctx.Status(201).JSON(fiber.Map{
		"message": "Comment added",
		"comment": comment,
	})
}

// GetTrending - Lấy video trending
// GET /api/videos/trending
func (c *VideoController) GetTrending(ctx *fiber.Ctx) error {
	limit := ctx.QueryInt("limit", 10)

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	findOptions := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "views", Value: -1}})

	cursor, err := services.VideosCollection.Find(dbCtx, bson.M{"status": "ready"}, findOptions)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch trending videos",
		})
	}
	defer cursor.Close(dbCtx)

	var videos []models.Video
	if err := cursor.All(dbCtx, &videos); err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to decode videos",
		})
	}

	return ctx.JSON(videos)
}

// GetLatest - Lấy video mới nhất
// GET /api/videos/latest
func (c *VideoController) GetLatest(ctx *fiber.Ctx) error {
	limit := ctx.QueryInt("limit", 10)

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	findOptions := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := services.VideosCollection.Find(dbCtx, bson.M{"status": "ready"}, findOptions)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch latest videos",
		})
	}
	defer cursor.Close(dbCtx)

	var videos []models.Video
	if err := cursor.All(dbCtx, &videos); err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to decode videos",
		})
	}

	return ctx.JSON(videos)
}

// ============ TELEGRAM CHANNEL SYNC METHODS ============

// SyncTelegramChannelStart - Bắt đầu sync nền, trả về ngay HTTP 202
// POST /api/videos/telegram/sync/start
func (c *VideoController) SyncTelegramChannelStart(ctx *fiber.Ctx) error {
	userID, err := middleware.RequireUserID(ctx)
	if err != nil {
		return err
	}
	userObjID, _ := primitive.ObjectIDFromHex(userID)

	if c.telegram == nil {
		return ctx.Status(500).JSON(fiber.Map{"error": "Telegram service not initialized"})
	}

	// Check if sync already running (our state or scanner)
	if c.telegramSync.getRunning() || c.telegram.IsScanning() {
		return ctx.Status(409).JSON(fiber.Map{
			"error": "Sync already in progress",
		})
	}

	// Ensure Telegram connection
	if !c.telegram.IsConnected() {
		if startErr := c.telegram.StartPersistentConnection(context.Background()); startErr != nil {
			return ctx.Status(500).JSON(fiber.Map{
				"error": "Telegram not connected: " + startErr.Error(),
			})
		}
		if !c.telegram.IsConnected() {
			return ctx.Status(500).JSON(fiber.Map{"error": "Telegram connection failed"})
		}
	}

	var req models.TelegramSyncRequest
	_ = ctx.BodyParser(&req)

	syncID := fmt.Sprintf("sync_%d", time.Now().UnixNano())
	c.telegramSync.setRunning(syncID)

	go c.runTelegramSyncBackground(userObjID, req)

	return ctx.Status(202).JSON(fiber.Map{
		"started": true,
		"sync_id": syncID,
	})
}

// getRunning returns whether a sync is currently running (helper for start handler)
func (s *telegramSyncState) getRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.IsRunning
}

// runTelegramSyncBackground - Worker chạy sync trong goroutine
func (c *VideoController) runTelegramSyncBackground(userObjID primitive.ObjectID, req models.TelegramSyncRequest) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[Sync] PANIC recovered: %v\n%s\n", r, debug.Stack())
			c.telegramSync.setDone(fmt.Sprintf("panic: %v", r))
		}
	}()

	// Get min message ID from DB
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	var lastDoc struct {
		TelegramMessageID int `bson:"telegram_message_id"`
	}
	_ = services.TelegramChannelVideosCollection.FindOne(
		dbCtx, bson.M{},
		options.FindOne().SetSort(bson.D{{Key: "telegram_message_id", Value: -1}}),
	).Decode(&lastDoc)
	cancel()
	minMsgID := lastDoc.TelegramMessageID

	scanCtx, scanCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer scanCancel()

	videos, err := c.telegram.ScanChannelInConnection(scanCtx, telegram.ScanOptions{
		ChannelID: req.ChannelID,
		Limit:     req.Limit,
		OffsetID:  req.OffsetID,
		MinMsgID:  minMsgID,
	})

	if err != nil {
		fmt.Printf("[Sync] ScanChannelInConnection failed: %v\n", err)
		c.telegramSync.setDone(err.Error())
		return
	}

	// Get existing IDs
	existingMsgIDs := make(map[int]bool)
	if len(videos) > 0 {
		msgIDs := make([]int, len(videos))
		for i, v := range videos {
			msgIDs[i] = v.MsgID
		}
		cursor, curErr := services.TelegramChannelVideosCollection.Find(
			context.Background(),
			bson.M{"telegram_message_id": bson.M{"$in": msgIDs}},
			options.Find().SetProjection(bson.M{"telegram_message_id": 1}),
		)
		if curErr == nil {
			for cursor.Next(context.Background()) {
				var doc struct{ TelegramMessageID int `bson:"telegram_message_id"` }
				if cursor.Decode(&doc) == nil {
					existingMsgIDs[doc.TelegramMessageID] = true
				}
			}
			cursor.Close(context.Background())
		}
	}

	total := len(videos)
	newCount := 0
	now := time.Now()

	for i, video := range videos {
		if existingMsgIDs[video.MsgID] {
			c.telegramSync.setProgress(i+1, total, newCount)
			continue
		}

		title := video.Text
		if title == "" {
			title = video.Name
		}
		if title == "" {
			title = fmt.Sprintf("Video %d", video.MsgID)
		}
		if len(title) > 200 {
			title = title[:200]
		}

		durationType := "short"
		if video.Dur > 300 {
			durationType = "medium"
		}
		if video.Dur > 600 {
			durationType = "long"
		}

		thumbnailURL := ""
		if video.ThumbID != "" && c.telegram != nil && c.cloudinary != nil {
			thumbPath, thumbErr := downloadTelegramThumbnail(context.Background(), c.telegram, video.ChanID, video.MsgID)
			if thumbErr == nil && thumbPath != "" {
				thumbResult, uploadErr := c.cloudinary.UploadImageFromPath(context.Background(), thumbPath, "video_thumbnails")
				if uploadErr == nil && thumbResult.SecureURL != "" {
					thumbnailURL = thumbResult.SecureURL
				}
				os.Remove(thumbPath)
			}
		}

		videoID := primitive.NewObjectID()
		mainVideo := models.Video{
			ID:                videoID,
			Title:             title,
			Description:       video.Text,
			Thumbnail:         thumbnailURL,
			StorageProvider:   models.StorageProviderTelegram,
			TelegramChannelID: video.ChanID,
			TelegramMessageID: video.MsgID,
			TelegramFileID:    fmt.Sprintf("%d", video.DocID),
			MimeType:          video.Mime,
			Duration:          video.Dur,
			DurationType:      durationType,
			FileSize:          video.Size,
			Width:             video.W,
			Height:            video.H,
			Status:            "ready",
			UploadedBy:        userObjID,
			CreatedAt:         now,
			UpdatedAt:         now,
		}

		_, insErr := services.VideosCollection.InsertOne(context.Background(), mainVideo)
		if insErr != nil {
			c.telegramSync.setProgress(i+1, total, newCount)
			continue
		}

		telegramVideo := models.TelegramChannelVideo{
			TelegramMessageID:   video.MsgID,
			TelegramGroupedID:   video.GrpID,
			TelegramChannelID:   video.ChanID,
			TelegramFileID:      fmt.Sprintf("%d", video.DocID),
			TelegramFileRef:     video.FileRef,
			TelegramAccessHash:  video.DocAccessHash,
			Caption:             video.Text,
			Duration:            video.Dur,
			FileSize:            video.Size,
			Width:               video.W,
			Height:              video.H,
			MimeType:            video.Mime,
			FileName:            video.Name,
			TelegramMessageDate: video.MsgDate,
			SyncedAt:            now,
			HasSpoiler:          video.Spoiler,
			SupportsStreaming:   video.StreamingSupport,
			IsPublished:         true,
			PublishedVideoID:    videoID,
		}

		_, insErr = services.TelegramChannelVideosCollection.InsertOne(context.Background(), telegramVideo)
		if insErr != nil {
			if mongo.IsDuplicateKeyError(insErr) {
				services.VideosCollection.DeleteOne(context.Background(), bson.M{"_id": videoID})
			}
		} else {
			newCount++
		}

		c.telegramSync.setProgress(i+1, total, newCount)
	}

	c.telegramSync.setDone("")
	fmt.Printf("[Sync] Background sync completed: %d new videos\n", newCount)
}

// SyncTelegramChannel - Đồng bộ video từ Telegram channel vào database và tự động publish (đồng bộ, giữ để tương thích)
// POST /api/videos/telegram/sync
func (c *VideoController) SyncTelegramChannel(ctx *fiber.Ctx) error {
	// Get user ID for auto-publish (safe)
	userID := ""
	if v := ctx.Locals("userID"); v != nil {
		if s, ok := v.(string); ok {
			userID = s
		}
	}
	if userID == "" {
		fmt.Println("[Sync] ERROR: Missing authorization header or invalid session")
		return ctx.Status(401).JSON(fiber.Map{"error": "Missing authorization header or unauthorized"})
	}
	userObjID, _ := primitive.ObjectIDFromHex(userID)

	// Panic recovery and logging
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[Sync] PANIC recovered: %v\n%s\n", r, debug.Stack())
		}
	}()

	if c.telegram == nil {
		fmt.Println("[Sync] ERROR: Telegram service not initialized")
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Telegram service not initialized",
		})
	}

	if !c.telegram.IsConnected() {
		// Try to start persistent connection automatically
		fmt.Println("[Sync] Telegram service not connected - attempting to start persistent connection...")
		startErr := c.telegram.StartPersistentConnection(context.Background())
		if startErr != nil {
			fmt.Printf("[Sync] Failed to start persistent connection: %v\n", startErr)
			return ctx.Status(500).JSON(fiber.Map{
				"error": "Telegram service not connected and failed to start connection: " + startErr.Error(),
			})
		}

		// Wait briefly for connected state
		if !c.telegram.IsConnected() {
			fmt.Println("[Sync] Telegram still not connected after start attempt")
			return ctx.Status(500).JSON(fiber.Map{
				"error": "Telegram service not connected after start attempt",
			})
		}
		fmt.Println("[Sync] Telegram persistent connection started successfully")
	}

	// Check if already scanning
	if c.telegram.IsScanning() {
		return ctx.Status(409).JSON(fiber.Map{
			"error":  "Sync already in progress",
			"status": c.telegram.GetScanStatus(),
		})
	}

	// Parse request body (optional)
	var req models.TelegramSyncRequest
	ctx.BodyParser(&req)

	// Get minimum message ID from database to avoid duplicates
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var minMsgID int
	var lastVideo models.TelegramChannelVideo
	err := services.TelegramChannelVideosCollection.FindOne(
		dbCtx,
		bson.M{},
		options.FindOne().SetSort(bson.D{{Key: "telegram_message_id", Value: -1}}),
	).Decode(&lastVideo)
	if err == nil {
		minMsgID = lastVideo.TelegramMessageID
	}

	fmt.Printf("[Sync] Starting sync - MinMsgID: %d, Limit: %d\n", minMsgID, req.Limit)

	// Start scanning in connection
	startTime := time.Now()
	scanCtx, scanCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer scanCancel()

	videos, err := c.telegram.ScanChannelInConnection(scanCtx, telegram.ScanOptions{
		ChannelID: req.ChannelID,
		Limit:     req.Limit,
		OffsetID:  req.OffsetID,
		MinMsgID:  minMsgID,
	})

	if err != nil {
		fmt.Printf("[Sync] ERROR: ScanChannelInConnection failed: %v\n", err)
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to scan channel: " + err.Error(),
		})
	}

	// Get existing message IDs to check for duplicates
	existingMsgIDs := make(map[int]bool)
	if len(videos) > 0 {
		msgIDs := make([]int, len(videos))
		for i, v := range videos {
			msgIDs[i] = v.MsgID
		}

		cursor, err := services.TelegramChannelVideosCollection.Find(
			context.Background(),
			bson.M{"telegram_message_id": bson.M{"$in": msgIDs}},
			options.Find().SetProjection(bson.M{"telegram_message_id": 1}),
		)
		if err == nil {
			defer cursor.Close(context.Background())
			for cursor.Next(context.Background()) {
				var doc struct {
					TelegramMessageID int `bson:"telegram_message_id"`
				}
				if cursor.Decode(&doc) == nil {
					existingMsgIDs[doc.TelegramMessageID] = true
				}
			}
		}
	}

	fmt.Printf("[Sync] Found %d existing videos in DB, %d scanned from Telegram\n", len(existingMsgIDs), len(videos))

	// Save only NEW videos to database AND auto-publish to Videos collection
	newCount := 0
	skippedCount := 0
	now := time.Now()

	for _, video := range videos {
		// Skip if already exists
		if existingMsgIDs[video.MsgID] {
			skippedCount++
			continue
		}

		// Generate title from caption or filename
		title := video.Text
		if title == "" {
			title = video.Name
		}
		if title == "" {
			title = fmt.Sprintf("Video %d", video.MsgID)
		}
		// Truncate title if too long (max 200 chars)
		if len(title) > 200 {
			title = title[:200]
		}

		// Determine duration type
		durationType := "short"
		if video.Dur > 300 {
			durationType = "medium"
		}
		if video.Dur > 600 {
			durationType = "long"
		}

		// Automatic thumbnail extraction from Telegram
		thumbnailURL := ""
		if video.ThumbID != "" && c.telegram != nil && c.cloudinary != nil {
			thumbPath, thumbErr := downloadTelegramThumbnail(context.Background(), c.telegram, video.ChanID, video.MsgID)
			if thumbErr == nil && thumbPath != "" {
				thumbResult, uploadErr := c.cloudinary.UploadImageFromPath(context.Background(), thumbPath, "video_thumbnails")
				if uploadErr == nil && thumbResult.SecureURL != "" {
					thumbnailURL = thumbResult.SecureURL
					fmt.Printf("[Sync] Uploaded thumbnail for video %d: %s\n", video.MsgID, thumbnailURL)
				}
				os.Remove(thumbPath)
			} else if thumbErr != nil {
				fmt.Printf("[Sync] Failed to download thumbnail for video %d: %v\n", video.MsgID, thumbErr)
			}
		}

		// Create main Video entry (auto-publish)
		videoID := primitive.NewObjectID()
		mainVideo := models.Video{
			ID:                videoID,
			Title:             title,
			Description:       video.Text,
			Thumbnail:         thumbnailURL,
			StorageProvider:   models.StorageProviderTelegram,
			TelegramChannelID: video.ChanID,
			TelegramMessageID: video.MsgID,
			TelegramFileID:    fmt.Sprintf("%d", video.DocID),
			MimeType:          video.Mime,
			Duration:          video.Dur,
			DurationType:      durationType,
			FileSize:          video.Size,
			Width:             video.W,
			Height:            video.H,
			Status:            "ready",
			UploadedBy:        userObjID,
			CreatedAt:         now,
			UpdatedAt:         now,
		}

		// Insert main video first
		_, err := services.VideosCollection.InsertOne(context.Background(), mainVideo)
		if err != nil {
			fmt.Printf("[Sync] Failed to create main video for message %d: %v\n", video.MsgID, err)
			continue
		}

		// Create TelegramChannelVideo record (for reference/tracking)
		telegramVideo := models.TelegramChannelVideo{
			TelegramMessageID:   video.MsgID,
			TelegramGroupedID:   video.GrpID,
			TelegramChannelID:   video.ChanID,
			TelegramFileID:      fmt.Sprintf("%d", video.DocID),
			TelegramFileRef:     video.FileRef,
			TelegramAccessHash:  video.DocAccessHash,
			Caption:             video.Text,
			Duration:            video.Dur,
			FileSize:            video.Size,
			Width:               video.W,
			Height:              video.H,
			MimeType:            video.Mime,
			FileName:            video.Name,
			TelegramMessageDate: video.MsgDate,
			SyncedAt:            now,
			HasSpoiler:          video.Spoiler,
			SupportsStreaming:   video.StreamingSupport,
			IsPublished:         true,
			PublishedVideoID:    videoID,
		}

		// Insert telegram video record
		_, err = services.TelegramChannelVideosCollection.InsertOne(
			context.Background(),
			telegramVideo,
		)
		if err != nil {
			if mongo.IsDuplicateKeyError(err) {
				// Rollback main video
				services.VideosCollection.DeleteOne(context.Background(), bson.M{"_id": videoID})
				skippedCount++
			} else {
				fmt.Printf("[Sync] Failed to save telegram video %d: %v\n", video.MsgID, err)
			}
		} else {
			newCount++
			fmt.Printf("[Sync] Auto-published video: %s (ID: %s)\n", title, videoID.Hex())
		}
	}

	duration := time.Since(startTime)
	status := c.telegram.GetScanStatus()

	return ctx.JSON(fiber.Map{
		"success":          true,
		"message":          fmt.Sprintf("Synced %d new videos from Telegram channel", newCount),
		"new_videos_count": newCount,
		"skipped_count":    skippedCount,
		"total_scanned":    status.TotalScanned,
		"sync_duration":    duration.String(),
	})
}

// downloadTelegramThumbnail - Download thumbnail từ Telegram, lưu vào temp file
// Returns: path to temp file, error
func downloadTelegramThumbnail(ctx context.Context, tgService *telegram.TelegramService, channelID int64, messageID int) (string, error) {
	if tgService == nil {
		return "", fmt.Errorf("Telegram service is nil")
	}

	var tempPath string
	var resultErr error

	err := tgService.ExecuteInConnection(ctx, func(execCtx context.Context) error {
		client := tgService.Client()
		api := client.GetAPI()

		messages, err := api.ChannelsGetMessages(execCtx, &tg.ChannelsGetMessagesRequest{
			Channel: &tg.InputChannel{ChannelID: channelID, AccessHash: client.GetAccessHash()},
			ID:      []tg.InputMessageClass{&tg.InputMessageID{ID: messageID}},
		})
		if err != nil {
			resultErr = fmt.Errorf("failed to get message: %w", err)
			return resultErr
		}

		channelMessages, ok := messages.(*tg.MessagesChannelMessages)
		if !ok || len(channelMessages.Messages) == 0 {
			resultErr = fmt.Errorf("message not found")
			return resultErr
		}

		msg, ok := channelMessages.Messages[0].(*tg.Message)
		if !ok {
			resultErr = fmt.Errorf("message not found")
			return resultErr
		}

		media, ok := msg.Media.(*tg.MessageMediaDocument)
		if !ok {
			resultErr = fmt.Errorf("no document media")
			return resultErr
		}

		doc, ok := media.Document.(*tg.Document)
		if !ok {
			resultErr = fmt.Errorf("no document")
			return resultErr
		}

		if len(doc.Thumbs) == 0 {
			resultErr = fmt.Errorf("no thumbnail available")
			return resultErr
		}

		thumb := doc.Thumbs[0]

		// Extract thumb type and approximate size from concrete types
		var thumbType string
		var thumbLimit int
		switch t := thumb.(type) {
		case *tg.PhotoSize:
			thumbType = t.Type
			thumbLimit = int(t.Size)
		case *tg.PhotoCachedSize:
			thumbType = t.Type
			thumbLimit = len(t.Bytes)
		default:
			// Fallback: request a small limit or full document size
			thumbType = ""
			if doc.Size > 0 {
				thumbLimit = int(doc.Size)
			} else {
				thumbLimit = 64 * 1024 // 64KB
			}
		}

		thumbLocation := &tg.InputDocumentFileLocation{
			ID:            doc.ID,
			AccessHash:    doc.AccessHash,
			FileReference: doc.FileReference,
			ThumbSize:     thumbType,
		}

		file, err := api.UploadGetFile(execCtx, &tg.UploadGetFileRequest{
			Location:     thumbLocation,
			Offset:       0,
			Limit:        thumbLimit,
			Precise:      true,
			CDNSupported: false,
		})
		if err != nil {
			resultErr = fmt.Errorf("failed to download thumbnail: %w", err)
			return resultErr
		}

		fileResult, ok := file.(*tg.UploadFile)
		if !ok {
			resultErr = fmt.Errorf("unexpected file result")
			return resultErr
		}

		tempDir := os.TempDir()
		tempPath = filepath.Join(tempDir, fmt.Sprintf("tg_thumb_%d_%d.jpg", channelID, messageID))
		err = os.WriteFile(tempPath, fileResult.Bytes, 0644)
		if err != nil {
			resultErr = fmt.Errorf("failed to save thumbnail: %w", err)
			return resultErr
		}

		return nil
	})

	if resultErr != nil {
		return "", resultErr
	}
	if err != nil {
		return "", err
	}

	return tempPath, nil
}

// GetSyncStatus - Lấy trạng thái đồng bộ hiện tại (async sync job)
// GET /api/videos/telegram/sync/status
func (c *VideoController) GetSyncStatus(ctx *fiber.Ctx) error {
	if c.telegram == nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Telegram service not initialized",
		})
	}

	running, syncID, progress, total, newCount, lastErr := c.telegramSync.get()
	resp := fiber.Map{
		"is_running":       running,
		"sync_id":          syncID,
		"current_progress": progress,
		"total":            total,
		"new_count":        newCount,
	}
	if lastErr != "" {
		resp["last_error"] = lastErr
	} else {
		resp["last_error"] = nil
	}
	return ctx.JSON(resp)
}

// GetTelegramVideos - Lấy danh sách video đã đồng bộ từ Telegram
// GET /api/videos/telegram/list
func (c *VideoController) GetTelegramVideos(ctx *fiber.Ctx) error {
	page := ctx.QueryInt("page", 1)
	limit := ctx.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100
	}
	skip := (page - 1) * limit

	// Build filter
	filter := bson.M{}
	if published := ctx.Query("published"); published != "" {
		filter["is_published"] = published == "true"
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Count total
	total, err := services.TelegramChannelVideosCollection.CountDocuments(dbCtx, filter)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to count videos",
		})
	}

	// Find videos
	findOptions := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "telegram_message_date", Value: -1}})

	cursor, err := services.TelegramChannelVideosCollection.Find(dbCtx, filter, findOptions)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch videos",
		})
	}
	defer cursor.Close(dbCtx)

	var videos []models.TelegramChannelVideo
	if err := cursor.All(dbCtx, &videos); err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to decode videos",
		})
	}

	totalPages := int(float64(total) / float64(limit))
	if total%int64(limit) > 0 {
		totalPages++
	}

	return ctx.JSON(models.TelegramChannelVideoListResponse{
		Videos:     videos,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	})
}

// PublishTelegramVideo - Publish video từ Telegram thành Video chính
// POST /api/videos/telegram/:id/publish
func (c *VideoController) PublishTelegramVideo(ctx *fiber.Ctx) error {
	userID, err := middleware.RequireUserID(ctx)
	if err != nil {
		return err
	}
	userObjID, _ := primitive.ObjectIDFromHex(userID)

	id := ctx.Params("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid video ID",
		})
	}

	// Parse request body
	var req struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
		Genres      []string `json:"genres"`
	}
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Title == "" {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Title is required",
		})
	}

	// Find Telegram video
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var telegramVideo models.TelegramChannelVideo
	err = services.TelegramChannelVideosCollection.FindOne(dbCtx, bson.M{"_id": objID}).Decode(&telegramVideo)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{
			"error": "Telegram video not found",
		})
	}

	if telegramVideo.IsPublished {
		return ctx.Status(400).JSON(fiber.Map{
			"error":    "Video already published",
			"video_id": telegramVideo.PublishedVideoID.Hex(),
		})
	}

	// Determine duration type
	durationType := "short"
	if telegramVideo.Duration > 300 {
		durationType = "medium"
	}
	if telegramVideo.Duration > 600 {
		durationType = "long"
	}

	// Create main Video from TelegramChannelVideo
	now := time.Now()
	video := models.Video{
		ID:                primitive.NewObjectID(),
		Title:             req.Title,
		Description:       req.Description,
		Tags:              req.Tags,
		Genres:            req.Genres,
		StorageProvider:   models.StorageProviderTelegram,
		TelegramChannelID: telegramVideo.TelegramChannelID,
		TelegramMessageID: telegramVideo.TelegramMessageID,
		TelegramFileID:    telegramVideo.TelegramFileID,
		MimeType:          telegramVideo.MimeType,
		Duration:          telegramVideo.Duration,
		DurationType:      durationType,
		FileSize:          telegramVideo.FileSize,
		Width:             telegramVideo.Width,
		Height:            telegramVideo.Height,
		Status:            "ready",
		UploadedBy:        userObjID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// Insert video
	_, err = services.VideosCollection.InsertOne(dbCtx, video)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to create video: " + err.Error(),
		})
	}

	// Update Telegram video as published
	_, err = services.TelegramChannelVideosCollection.UpdateOne(
		dbCtx,
		bson.M{"_id": objID},
		bson.M{"$set": bson.M{
			"is_published":       true,
			"published_video_id": video.ID,
		}},
	)
	if err != nil {
		fmt.Printf("[Publish] Warning: Failed to update telegram video status: %v\n", err)
	}

	return ctx.Status(201).JSON(fiber.Map{
		"message": "Video published successfully",
		"video":   video,
	})
}

// StreamTelegramChannelVideo - Stream video trực tiếp từ Telegram channel video
// GET /api/videos/telegram/:id/stream
func (c *VideoController) StreamTelegramChannelVideo(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid video ID",
		})
	}

	// Find Telegram video
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var telegramVideo models.TelegramChannelVideo
	err = services.TelegramChannelVideosCollection.FindOne(dbCtx, bson.M{"_id": objID}).Decode(&telegramVideo)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{
			"error": "Video not found",
		})
	}

	// Check Telegram connection
	if c.telegram == nil || !c.telegram.IsConnected() {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Telegram service not connected",
		})
	}

	// Get file size
	fileSize := telegramVideo.FileSize
	if fileSize == 0 {
		size, err := c.telegram.GetFileSize(context.Background(), telegramVideo.TelegramMessageID)
		if err != nil {
			return ctx.Status(500).JSON(fiber.Map{
				"error": "Failed to get file size: " + err.Error(),
			})
		}
		fileSize = size
	}

	// Parse Range header
	rangeHeader := ctx.Get("Range")
	start, end, err := telegram.ParseRangeHeader(rangeHeader, fileSize)
	if err != nil {
		return ctx.Status(416).JSON(fiber.Map{
			"error": "Range not satisfiable",
		})
	}

	// Determine content type
	contentType := telegramVideo.MimeType
	if contentType == "" {
		contentType = "video/mp4"
	}

	// Set response headers
	ctx.Set("Content-Type", contentType)
	ctx.Set("Accept-Ranges", "bytes")
	ctx.Set("Cache-Control", "public, max-age=3600")

	if rangeHeader != "" {
		ctx.Set("Content-Length", fmt.Sprintf("%d", end-start+1))
		ctx.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
		ctx.Status(206)
	} else {
		ctx.Set("Content-Length", fmt.Sprintf("%d", fileSize))
		ctx.Status(200)
	}

	// Stream video content
	ctx.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		err := c.telegram.StreamVideo(context.Background(), telegram.StreamRequest{
			ChannelID: telegramVideo.TelegramChannelID,
			MessageID: telegramVideo.TelegramMessageID,
			FileID:    telegramVideo.TelegramFileID,
			Start:     start,
			End:       end,
		}, w)
		if err != nil {
			fmt.Printf("[StreamTelegramChannel] ERROR: %v\n", err)
		}
		w.Flush()
	})

	return nil
}

// DeleteTelegramVideo - Xóa video đã đồng bộ (không xóa trên Telegram)
// DELETE /api/videos/telegram/:id
func (c *VideoController) DeleteTelegramVideo(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid video ID",
		})
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := services.TelegramChannelVideosCollection.DeleteOne(dbCtx, bson.M{"_id": objID})
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to delete video",
		})
	}

	if result.DeletedCount == 0 {
		return ctx.Status(404).JSON(fiber.Map{
			"error": "Video not found",
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Video deleted successfully",
	})
}
