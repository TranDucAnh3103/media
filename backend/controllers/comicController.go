package controllers

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"media-backend/models"
	"media-backend/services"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ComicController - Controller xử lý comic operations
type ComicController struct {
	cloudinary *services.CloudinaryService
}

// NewComicController - Tạo instance mới
func NewComicController() *ComicController {
	cloudinary, _ := services.NewCloudinaryService()
	return &ComicController{cloudinary: cloudinary}
}

// GetComics - Lấy danh sách truyện với filter & pagination
// GET /api/comics
func (c *ComicController) GetComics(ctx *fiber.Ctx) error {
	// Parse query params
	page := ctx.QueryInt("page", 1)
	limit := ctx.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100
	}
	skip := (page - 1) * limit

	// Build filter
	filter := bson.M{}

	if genres := ctx.Query("genres"); genres != "" {
		filter["genres"] = bson.M{"$in": strings.Split(genres, ",")}
	}
	if status := ctx.Query("status"); status != "" {
		filter["status"] = status
	}
	if author := ctx.Query("author"); author != "" {
		filter["author"] = bson.M{"$regex": author, "$options": "i"}
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
	total, err := services.ComicsCollection.CountDocuments(dbCtx, filter)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to count comics",
		})
	}

	// Find comics
	findOptions := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: sortField, Value: sortOrder}})

	cursor, err := services.ComicsCollection.Find(dbCtx, filter, findOptions)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch comics",
		})
	}
	defer cursor.Close(dbCtx)

	var comics []models.Comic
	if err := cursor.All(dbCtx, &comics); err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to decode comics",
		})
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return ctx.JSON(models.ComicListResponse{
		Comics:     comics,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	})
}

// GetComic - Lấy chi tiết một truyện
// GET /api/comics/:id
func (c *ComicController) GetComic(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid comic ID",
		})
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var comic models.Comic
	err = services.ComicsCollection.FindOne(dbCtx, bson.M{"_id": objID}).Decode(&comic)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{
			"error": "Comic not found",
		})
	}

	// Tăng views
	go func() {
		updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		services.ComicsCollection.UpdateOne(updateCtx, bson.M{"_id": objID}, bson.M{
			"$inc": bson.M{"views": 1},
		})
	}()

	return ctx.JSON(comic)
}

// CreateComic - Tạo truyện mới
// POST /api/comics
func (c *ComicController) CreateComic(ctx *fiber.Ctx) error {
	userID := ctx.Locals("userID").(string)
	userObjID, _ := primitive.ObjectIDFromHex(userID)

	var req models.ComicCreateRequest
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

	now := time.Now()
	comic := models.Comic{
		ID:          primitive.NewObjectID(),
		Title:       req.Title,
		Description: req.Description,
		Author:      req.Author,
		Tags:        req.Tags,
		Genres:      req.Genres,
		Status:      req.Status,
		Chapters:    []models.Chapter{},
		Views:       0,
		Likes:       0,
		UploadedBy:  userObjID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if comic.Status == "" {
		comic.Status = "ongoing"
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := services.ComicsCollection.InsertOne(dbCtx, comic)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to create comic",
		})
	}

	return ctx.Status(201).JSON(fiber.Map{
		"message": "Comic created successfully",
		"comic":   comic,
	})
}

// GetMyComics - Lấy danh sách truyện của user hiện tại
// GET /api/comics/my
// Đối với app cá nhân: trả về TẤT CẢ truyện
func (c *ComicController) GetMyComics(ctx *fiber.Ctx) error {
	// Xác thực user (không filter theo user vì đây là app cá nhân)
	_ = ctx.Locals("userID").(string)

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Lấy tất cả truyện (app cá nhân - không cần phân quyền)
	cursor, err := services.ComicsCollection.Find(dbCtx, bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch comics",
		})
	}
	defer cursor.Close(dbCtx)

	var comics []models.Comic
	if err := cursor.All(dbCtx, &comics); err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to decode comics",
		})
	}

	return ctx.JSON(fiber.Map{
		"comics": comics,
	})
}

// UploadComicWithImages - Upload truyện mới với ảnh (đơn giản hóa - không cần chapter riêng)
// POST /api/comics/upload
func (c *ComicController) UploadComicWithImages(ctx *fiber.Ctx) error {
	userID := ctx.Locals("userID").(string)
	userObjID, _ := primitive.ObjectIDFromHex(userID)

	if c.cloudinary == nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Cloudinary service not available",
		})
	}

	// Parse form data
	title := ctx.FormValue("title")
	description := ctx.FormValue("description")
	if description == "" {
		description = "Truyện hành động"
	}

	if title == "" {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Title is required",
		})
	}

	// Parse multipart form
	form, err := ctx.MultipartForm()
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid form data",
		})
	}

	imageFiles := form.File["images"]
	if len(imageFiles) == 0 {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "At least one image is required",
		})
	}

	// Create comic first
	now := time.Now()
	comicID := primitive.NewObjectID()

	// Upload images to Cloudinary - lưu trực tiếp trong folder comics/{comicID}
	folder := fmt.Sprintf("comics/%s", comicID.Hex())
	results, err := c.cloudinary.UploadImages(ctx.Context(), imageFiles, folder)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to upload images: %v", err),
		})
	}

	var uploadedImages []models.ComicImage
	for i, result := range results {
		uploadedImages = append(uploadedImages, models.ComicImage{
			Page:     i + 1,
			URL:      result.SecureURL,
			PublicID: result.PublicID,
			Width:    result.Width,
			Height:   result.Height,
		})
	}

	// Create chapter with uploaded images
	chapter := models.Chapter{
		ID:         primitive.NewObjectID(),
		Number:     1,
		Title:      "Chapter 1",
		Images:     uploadedImages,
		Views:      0,
		UploadedAt: now,
	}

	// Set cover image as first image
	coverImage := ""
	if len(uploadedImages) > 0 {
		coverImage = uploadedImages[0].URL
	}

	comic := models.Comic{
		ID:          comicID,
		Title:       title,
		Description: description,
		Author:      "",
		CoverImage:  coverImage,
		Tags:        []string{"action"},
		Genres:      []string{"Action"},
		Status:      "ongoing",
		Chapters:    []models.Chapter{chapter},
		Views:       0,
		Likes:       0,
		UploadedBy:  userObjID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = services.ComicsCollection.InsertOne(dbCtx, comic)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to create comic",
		})
	}

	return ctx.Status(201).JSON(fiber.Map{
		"message":      "Comic uploaded successfully",
		"comic":        comic,
		"images_count": len(uploadedImages),
	})
}

// UpdateComic - Cập nhật truyện
// PUT /api/comics/:id
func (c *ComicController) UpdateComic(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid comic ID",
		})
	}

	var req models.ComicUpdateRequest
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
	if req.Author != "" {
		update["$set"].(bson.M)["author"] = req.Author
	}
	if len(req.Tags) > 0 {
		update["$set"].(bson.M)["tags"] = req.Tags
	}
	if len(req.Genres) > 0 {
		update["$set"].(bson.M)["genres"] = req.Genres
	}
	if req.Status != "" {
		update["$set"].(bson.M)["status"] = req.Status
	}
	if req.CoverImage != "" {
		update["$set"].(bson.M)["cover_image"] = req.CoverImage
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := services.ComicsCollection.UpdateOne(dbCtx, bson.M{"_id": objID}, update)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to update comic",
		})
	}

	if result.MatchedCount == 0 {
		return ctx.Status(404).JSON(fiber.Map{
			"error": "Comic not found",
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Comic updated successfully",
	})
}

// DeleteComic - Xóa truyện
// DELETE /api/comics/:id
func (c *ComicController) DeleteComic(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid comic ID",
		})
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Lấy comic để xóa ảnh từ Cloudinary
	var comic models.Comic
	err = services.ComicsCollection.FindOne(dbCtx, bson.M{"_id": objID}).Decode(&comic)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{
			"error": "Comic not found",
		})
	}

	// Xóa folder chứa tất cả ảnh từ Cloudinary (background)
	go func() {
		if c.cloudinary == nil {
			return
		}
		deleteCtx := context.Background()
		folder := fmt.Sprintf("comics/%s", id)
		c.cloudinary.DeleteFolder(deleteCtx, folder, "image")
	}()

	// Xóa comic từ database
	result, err := services.ComicsCollection.DeleteOne(dbCtx, bson.M{"_id": objID})
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to delete comic",
		})
	}

	if result.DeletedCount == 0 {
		return ctx.Status(404).JSON(fiber.Map{
			"error": "Comic not found",
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Comic deleted successfully",
	})
}

// UploadChapter - Upload chapter mới (nhiều ảnh hoặc zip)
// POST /api/comics/:id/chapters
func (c *ComicController) UploadChapter(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid comic ID",
		})
	}

	if c.cloudinary == nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Cloudinary service not available",
		})
	}

	// Parse chapter info
	chapterNumber := ctx.FormValue("chapter_number", "1")
	chapterTitle := ctx.FormValue("title", "")

	var chapterNum int
	fmt.Sscanf(chapterNumber, "%d", &chapterNum)

	// Parse multipart form
	form, err := ctx.MultipartForm()
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid form data",
		})
	}

	var uploadedImages []models.ComicImage

	// Check if zip file uploaded
	zipFiles := form.File["zip"]
	if len(zipFiles) > 0 {
		// Process ZIP file
		images, err := c.processZipUpload(ctx.Context(), zipFiles[0], objID.Hex(), chapterNum)
		if err != nil {
			return ctx.Status(500).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to process zip: %v", err),
			})
		}
		uploadedImages = images
	} else {
		// Process multiple image files
		imageFiles := form.File["images"]
		if len(imageFiles) == 0 {
			return ctx.Status(400).JSON(fiber.Map{
				"error": "No images or zip file provided",
			})
		}

		folder := fmt.Sprintf("comics/%s/chapter_%d", objID.Hex(), chapterNum)
		results, err := c.cloudinary.UploadImages(ctx.Context(), imageFiles, folder)
		if err != nil {
			return ctx.Status(500).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to upload images: %v", err),
			})
		}

		for i, result := range results {
			uploadedImages = append(uploadedImages, models.ComicImage{
				Page:     i + 1,
				URL:      result.SecureURL,
				PublicID: result.PublicID,
				Width:    result.Width,
				Height:   result.Height,
			})
		}
	}

	// Create chapter
	chapter := models.Chapter{
		ID:         primitive.NewObjectID(),
		Number:     chapterNum,
		Title:      chapterTitle,
		Images:     uploadedImages,
		Views:      0,
		UploadedAt: time.Now(),
	}

	// Update comic with new chapter
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Remove existing chapter with same number if exists
	services.ComicsCollection.UpdateOne(dbCtx, bson.M{"_id": objID}, bson.M{
		"$pull": bson.M{"chapters": bson.M{"number": chapterNum}},
	})

	// Add new chapter
	_, err = services.ComicsCollection.UpdateOne(dbCtx, bson.M{"_id": objID}, bson.M{
		"$push": bson.M{"chapters": chapter},
		"$set":  bson.M{"updated_at": time.Now()},
	})
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to save chapter",
		})
	}

	// Set cover image if first chapter and no cover
	var comic models.Comic
	services.ComicsCollection.FindOne(dbCtx, bson.M{"_id": objID}).Decode(&comic)
	if comic.CoverImage == "" && len(uploadedImages) > 0 {
		services.ComicsCollection.UpdateOne(dbCtx, bson.M{"_id": objID}, bson.M{
			"$set": bson.M{"cover_image": uploadedImages[0].URL},
		})
	}

	return ctx.Status(201).JSON(fiber.Map{
		"message":      "Chapter uploaded successfully",
		"chapter":      chapter,
		"images_count": len(uploadedImages),
	})
}

// processZipUpload - Xử lý upload file zip (multi-thread với Goroutines)
func (c *ComicController) processZipUpload(ctx context.Context, fileHeader interface{}, comicID string, chapterNum int) ([]models.ComicImage, error) {
	// TODO: This is a simplified version. In production, handle multipart.FileHeader properly.
	// Extract zip to temp, upload images in parallel

	// For now, return empty - implement full zip processing
	return nil, fmt.Errorf("zip upload not implemented yet")
}

// extractZip - Giải nén file zip vào thư mục tạm
func extractZip(zipPath string, destDir string) ([]string, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var files []string
	imageExtensions := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	}

	for _, file := range reader.File {
		ext := strings.ToLower(filepath.Ext(file.Name))
		if !imageExtensions[ext] {
			continue
		}

		// Tạo file output
		outPath := filepath.Join(destDir, filepath.Base(file.Name))
		outFile, err := os.Create(outPath)
		if err != nil {
			continue
		}

		// Copy nội dung
		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			continue
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()

		if err == nil {
			files = append(files, outPath)
		}
	}

	// Sort files by name
	sort.Strings(files)

	return files, nil
}

// GetChapter - Lấy chapter của truyện
// GET /api/comics/:id/chapters/:chapterNum
func (c *ComicController) GetChapter(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	chapterNumStr := ctx.Params("chapterNum")

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid comic ID",
		})
	}

	var chapterNum int
	fmt.Sscanf(chapterNumStr, "%d", &chapterNum)

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var comic models.Comic
	err = services.ComicsCollection.FindOne(dbCtx, bson.M{"_id": objID}).Decode(&comic)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{
			"error": "Comic not found",
		})
	}

	// Find chapter
	for _, chapter := range comic.Chapters {
		if chapter.Number == chapterNum {
			// Tăng views cho chapter
			go func() {
				updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				services.ComicsCollection.UpdateOne(updateCtx,
					bson.M{"_id": objID, "chapters.number": chapterNum},
					bson.M{"$inc": bson.M{"chapters.$.views": 1}},
				)
			}()

			return ctx.JSON(chapter)
		}
	}

	return ctx.Status(404).JSON(fiber.Map{
		"error": "Chapter not found",
	})
}

// GetTrending - Lấy truyện trending
// GET /api/comics/trending
func (c *ComicController) GetTrending(ctx *fiber.Ctx) error {
	limit := ctx.QueryInt("limit", 10)

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	findOptions := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "views", Value: -1}})

	cursor, err := services.ComicsCollection.Find(dbCtx, bson.M{}, findOptions)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch trending comics",
		})
	}
	defer cursor.Close(dbCtx)

	var comics []models.Comic
	if err := cursor.All(dbCtx, &comics); err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to decode comics",
		})
	}

	return ctx.JSON(comics)
}

// GetLatest - Lấy truyện mới nhất
// GET /api/comics/latest
func (c *ComicController) GetLatest(ctx *fiber.Ctx) error {
	limit := ctx.QueryInt("limit", 10)

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	findOptions := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "updated_at", Value: -1}})

	cursor, err := services.ComicsCollection.Find(dbCtx, bson.M{}, findOptions)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch latest comics",
		})
	}
	defer cursor.Close(dbCtx)

	var comics []models.Comic
	if err := cursor.All(dbCtx, &comics); err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to decode comics",
		})
	}

	return ctx.JSON(comics)
}

// LikeComic - Like/Unlike comic (toggle)
// POST /api/comics/:id/like
func (c *ComicController) LikeComic(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	userID := ctx.Locals("userID").(string)

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid comic ID",
		})
	}

	userObjID, _ := primitive.ObjectIDFromHex(userID)

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if user already liked this comic
	var user models.User
	err = services.UsersCollection.FindOne(dbCtx, bson.M{"_id": userObjID}).Decode(&user)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Check if comic is already liked
	alreadyLiked := false
	for _, likedID := range user.LikedComics {
		if likedID == objID {
			alreadyLiked = true
			break
		}
	}

	if alreadyLiked {
		// Unlike: remove from user's liked list and decrement count
		_, err = services.UsersCollection.UpdateOne(dbCtx, bson.M{"_id": userObjID}, bson.M{
			"$pull": bson.M{"liked_comics": objID},
		})
		if err != nil {
			return ctx.Status(500).JSON(fiber.Map{
				"error": "Failed to unlike comic",
			})
		}

		_, err = services.ComicsCollection.UpdateOne(dbCtx, bson.M{"_id": objID}, bson.M{
			"$inc": bson.M{"likes": -1},
		})
		if err != nil {
			return ctx.Status(500).JSON(fiber.Map{
				"error": "Failed to update comic likes",
			})
		}

		return ctx.JSON(fiber.Map{
			"message": "Comic unliked",
			"liked":   false,
		})
	}

	// Like: add to user's liked list and increment count
	_, err = services.UsersCollection.UpdateOne(dbCtx, bson.M{"_id": userObjID}, bson.M{
		"$addToSet": bson.M{"liked_comics": objID},
	})
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to like comic",
		})
	}

	_, err = services.ComicsCollection.UpdateOne(dbCtx, bson.M{"_id": objID}, bson.M{
		"$inc": bson.M{"likes": 1},
	})
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to update comic likes",
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Comic liked",
		"liked":   true,
	})
}
