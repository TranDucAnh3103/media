package controllers

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"media-backend/models"
	"media-backend/services"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// VideoController - Controller xử lý video operations
type VideoController struct {
	cloudinary *services.CloudinaryService
	mega       *services.MegaService
	uploading  sync.Map // Track upload progress
}

// NewVideoController - Tạo instance mới
func NewVideoController() *VideoController {
	cloudinary, _ := services.NewCloudinaryService()
	mega, _ := services.NewMegaService()
	return &VideoController{
		cloudinary: cloudinary,
		mega:       mega,
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

// StreamMegaVideo - Stream video từ Mega
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

// GetMyVideos - Lấy danh sách video của user hiện tại
// GET /api/videos/my
func (c *VideoController) GetMyVideos(ctx *fiber.Ctx) error {
	userID := ctx.Locals("userID").(string)
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := services.VideosCollection.Find(dbCtx, bson.M{"uploaded_by": userObjID}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
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
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to create video record",
		})
	}

	// Upload video in background (multi-thread)
	go c.processVideoUpload(videoID, file, userID)

	return ctx.Status(202).JSON(fiber.Map{
		"message":  "Video upload started",
		"video_id": videoID.Hex(),
		"status":   "processing",
	})
}

// processVideoUpload - Xử lý upload video trong background (Goroutine)
func (c *VideoController) processVideoUpload(videoID primitive.ObjectID, file *multipart.FileHeader, userID string) {
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

	// Open file
	src, err := file.Open()
	if err != nil {
		c.updateVideoStatus(videoID, "error", "Failed to open file")
		return
	}
	defer src.Close()

	// Determine storage based on estimated duration
	// For simplicity, use file size as proxy (rough estimate: 1MB = ~10 seconds for compressed video)
	estimatedDuration := int(file.Size / (1024 * 1024) * 10) // seconds
	durationType := "short"
	if estimatedDuration > 300 {
		durationType = "medium"
	}
	if estimatedDuration > 600 {
		durationType = "long"
	}

	var videoURL, thumbnail, storageType string
	var duration int

	if durationType == "long" || file.Size > 100*1024*1024 { // > 100MB → Mega
		// Upload to Mega
		if c.mega == nil {
			c.updateVideoStatus(videoID, "error", "Mega service not available")
			return
		}

		c.uploading.Store(progressID, models.UploadProgress{
			ID:       progressID,
			Status:   "uploading",
			Progress: 20,
			Message:  "Uploading to Mega...",
		})

		// Save to temp file first
		tempDir := os.TempDir()
		tempPath := filepath.Join(tempDir, file.Filename)
		tempFile, err := os.Create(tempPath)
		if err != nil {
			c.updateVideoStatus(videoID, "error", "Failed to create temp file")
			return
		}

		_, err = tempFile.ReadFrom(src)
		tempFile.Close()
		if err != nil {
			os.Remove(tempPath)
			c.updateVideoStatus(videoID, "error", "Failed to save temp file")
			return
		}

		// Upload to Mega
		progressChan := make(chan services.MegaUploadProgress, 100)
		go func() {
			for p := range progressChan {
				c.uploading.Store(progressID, models.UploadProgress{
					ID:       progressID,
					Status:   "uploading",
					Progress: float64(p.Percent),
					Message:  fmt.Sprintf("Uploading to Mega: %d%%", p.Percent),
				})
			}
		}()

		result, err := c.mega.UploadFile(context.Background(), tempPath, "videos", progressChan)
		close(progressChan)
		os.Remove(tempPath)

		if err != nil {
			c.updateVideoStatus(videoID, "error", fmt.Sprintf("Mega upload failed: %v", err))
			return
		}

		videoURL = result.PublicURL
		storageType = "mega"
		duration = estimatedDuration

	} else {
		// Upload to Cloudinary
		if c.cloudinary == nil {
			c.updateVideoStatus(videoID, "error", "Cloudinary service not available")
			return
		}

		c.uploading.Store(progressID, models.UploadProgress{
			ID:       progressID,
			Status:   "uploading",
			Progress: 30,
			Message:  "Uploading to Cloudinary...",
		})

		result, err := c.cloudinary.UploadVideo(context.Background(), src, file.Filename, "videos")
		if err != nil {
			c.updateVideoStatus(videoID, "error", fmt.Sprintf("Cloudinary upload failed: %v", err))
			return
		}

		videoURL = result.SecureURL
		thumbnail = c.cloudinary.GenerateThumbnail(result.PublicID)
		storageType = "cloudinary"
		duration = result.Duration

		// Save Cloudinary PublicID for later deletion
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		services.VideosCollection.UpdateOne(ctx, bson.M{"_id": videoID}, bson.M{
			"$set": bson.M{"cloudinary_public_id": result.PublicID},
		})
		cancel()
	}

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
			"video_url":     videoURL,
			"thumbnail":     thumbnail,
			"storage_type":  storageType,
			"duration":      duration,
			"duration_type": durationType,
			"status":        "ready",
			"updated_at":    time.Now(),
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
		if video.StorageType == "cloudinary" && c.cloudinary != nil {
			// Xóa video từ Cloudinary bằng PublicID
			if video.CloudinaryPublicID != "" {
				c.cloudinary.DeleteResource(deleteCtx, video.CloudinaryPublicID, "video")
			}
		} else if video.StorageType == "mega" && c.mega != nil {
			// TODO: Implement mega delete
			// c.mega.DeleteFile(video.MegaFileHash)
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
