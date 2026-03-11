package controllers

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"media-backend/models"
	"media-backend/services"
	"media-backend/services/telegram"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// VideoController - Controller xử lý video operations
type VideoController struct {
	cloudinary *services.CloudinaryService
	mega       *services.MegaService     // Legacy - giữ lại cho backward compatibility
	telegram   *telegram.TelegramService // New - Telegram storage
	uploading  sync.Map                  // Track upload progress
}

// NewVideoController - Tạo instance mới
func NewVideoController() *VideoController {
	cloudinary, _ := services.NewCloudinaryService()
	mega, _ := services.NewMegaService() // Legacy

	// Initialize Telegram service
	tg, err := telegram.NewTelegramService()
	if err != nil {
		fmt.Printf("Warning: Failed to initialize Telegram service: %v\n", err)
	}

	return &VideoController{
		cloudinary: cloudinary,
		mega:       mega,
		telegram:   tg,
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
		size, err := c.telegram.GetFileSize(context.Background(), video.TelegramMessageID)
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

	ctx.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		err := c.telegram.StreamVideo(context.Background(), telegram.StreamRequest{
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
	_ = ctx.Locals("userID").(string)

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
	userID := ctx.Locals("userID").(string)
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
	userID := ctx.Locals("userID").(string)

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
	userID := ctx.Locals("userID").(string)

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
