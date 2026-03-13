package routes

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"media-backend/controllers"
	"media-backend/middleware"
	"media-backend/services"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SetupRoutes - Cấu hình tất cả routes cho API
func SetupRoutes(app *fiber.App) {
	// Kết nối database
	if err := services.ConnectDB(); err != nil {
		panic(err)
	}

	// Controllers
	userController := controllers.NewUserController()
	comicController := controllers.NewComicController()
	videoController := controllers.NewVideoController()

	// Start Telegram persistent connection - BLOCKING (required for video upload/streaming)
	fmt.Println("[Routes] Initializing Telegram connection...")
	ctx := context.Background()
	if err := videoController.InitTelegramConnection(ctx); err != nil {
		fmt.Printf("[Routes] ⚠️  Telegram connection failed: %v\n", err)
		fmt.Println("[Routes] ⚠️  Video upload and streaming will NOT work!")
		fmt.Println("[Routes] ⚠️  Check TELEGRAM_* environment variables in .env")
	} else {
		fmt.Println("[Routes] ✅ Telegram connection ready - video upload/streaming enabled")
	}

	// API group
	api := app.Group("/api")

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "Media API is running",
		})
	})

	// ============ AUTH ROUTES ============
	auth := api.Group("/auth")
	auth.Post("/register", userController.Register)
	auth.Post("/login", userController.Login)

	// ============ USER ROUTES (Protected) ============
	user := api.Group("/user", middleware.AuthMiddleware())
	user.Get("/profile", userController.GetProfile)
	user.Put("/profile", userController.UpdateProfile)
	user.Post("/bookmarks", userController.AddBookmark)
	user.Delete("/bookmarks/:contentId", userController.RemoveBookmark)
	user.Delete("/liked/:type/:contentId", userController.RemoveLike)
	user.Post("/playlists", userController.CreatePlaylist)
	user.Post("/playlists/:playlistId/videos", userController.AddToPlaylist)

	// ============ COMIC ROUTES ============
	comics := api.Group("/comics")

	// Protected routes that need auth (MUST come first before :id)
	comicsProtected := comics.Group("", middleware.AuthMiddleware())
	comicsProtected.Get("/my", comicController.GetMyComics)
	comicsProtected.Post("/upload", comicController.UploadComicWithImages)

	// Public routes (specific paths first)
	comics.Get("/trending", comicController.GetTrending)
	comics.Get("/latest", comicController.GetLatest)
	comics.Get("/", comicController.GetComics)
	comics.Get("/:id", comicController.GetComic)
	comics.Get("/:id/chapters/:chapterNum", comicController.GetChapter)

	// Protected routes for existing comics
	comicsProtected.Post("/", comicController.CreateComic)
	comicsProtected.Put("/:id", comicController.UpdateComic)
	comicsProtected.Delete("/:id", comicController.DeleteComic)
	comicsProtected.Post("/:id/chapters", comicController.UploadChapter)
	comicsProtected.Post("/:id/like", comicController.LikeComic)

	// ============ VIDEO ROUTES ============
	videos := api.Group("/videos")

	// IMPORTANT: Routes are matched in registration order!
	// Specific paths MUST be registered BEFORE dynamic :id routes

	// PUBLIC routes (no authentication required)
	videos.Get("/trending", videoController.GetTrending)
	videos.Get("/latest", videoController.GetLatest)
	videos.Get("/upload/progress/:id", videoController.GetUploadProgress)
	videos.Get("/telegram/status", videoController.GetTelegramStatus)              // Telegram status check
	videos.Get("/telegram/sync/status", videoController.GetSyncStatus)             // Sync status
	videos.Get("/telegram/list", videoController.GetTelegramVideos)                // List synced videos
	videos.Get("/telegram/:id/stream", videoController.StreamTelegramChannelVideo) // Stream synced video
	videos.Get("/stream/mega/:hash", videoController.StreamMegaVideo)              // Mega streaming (legacy)
	videos.Get("/stream/:id", videoController.StreamVideo)                         // Unified streaming (Telegram/Cloudinary/Mega)

	// PROTECTED Telegram sync routes (sync/start before sync for route precedence)
	videos.Post("/telegram/sync/start", middleware.AuthMiddleware(), videoController.SyncTelegramChannelStart) // Async sync - returns 202
	videos.Post("/telegram/sync", middleware.AuthMiddleware(), videoController.SyncTelegramChannel)            // Sync (legacy, blocking)
	videos.Post("/telegram/:id/publish", middleware.AuthMiddleware(), videoController.PublishTelegramVideo) // Publish video
	videos.Delete("/telegram/:id", middleware.AuthMiddleware(), videoController.DeleteTelegramVideo)        // Delete synced video

	// PROTECTED routes with specific paths (must come before /:id)
	videos.Get("/my", middleware.AuthMiddleware(), videoController.GetMyVideos)
	videos.Post("/upload", middleware.AuthMiddleware(), videoController.UploadVideo)

	// PUBLIC routes with dynamic :id (:id/thumbnail before :id for correct matching)
	videos.Get("/:id/thumbnail", videoController.GetVideoThumbnail)
	videos.Get("/", videoController.GetVideos)
	videos.Get("/:id", videoController.GetVideo)

	// PROTECTED routes with dynamic :id (different HTTP methods, so no conflict)
	videos.Put("/:id", middleware.AuthMiddleware(), videoController.UpdateVideo)
	videos.Delete("/:id", middleware.AuthMiddleware(), videoController.DeleteVideo)
	videos.Post("/:id/like", middleware.AuthMiddleware(), videoController.LikeVideo)
	videos.Post("/:id/comments", middleware.AuthMiddleware(), videoController.AddComment)

	// ============ ADMIN ROUTES ============
	admin := api.Group("/admin", middleware.AuthMiddleware())

	// Stats
	admin.Get("/stats", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		usersCount, _ := services.UsersCollection.CountDocuments(ctx, bson.M{})
		comicsCount, _ := services.ComicsCollection.CountDocuments(ctx, bson.M{})
		videosCount, _ := services.VideosCollection.CountDocuments(ctx, bson.M{})

		return c.JSON(fiber.Map{
			"users":  usersCount,
			"comics": comicsCount,
			"videos": videosCount,
		})
	})

	// Users
	admin.Get("/users", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cursor, err := services.UsersCollection.Find(ctx, bson.M{})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch users"})
		}
		defer cursor.Close(ctx)

		var users []bson.M
		if err := cursor.All(ctx, &users); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to decode users"})
		}

		// Remove passwords
		for i := range users {
			delete(users[i], "password")
		}

		return c.JSON(fiber.Map{"users": users})
	})

	// Videos
	admin.Get("/videos", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cursor, err := services.VideosCollection.Find(ctx, bson.M{})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch videos"})
		}
		defer cursor.Close(ctx)

		var videos []bson.M
		if err := cursor.All(ctx, &videos); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to decode videos"})
		}

		return c.JSON(fiber.Map{"videos": videos})
	})

	// Comics
	admin.Get("/comics", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cursor, err := services.ComicsCollection.Find(ctx, bson.M{})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch comics"})
		}
		defer cursor.Close(ctx)

		var comics []bson.M
		if err := cursor.All(ctx, &comics); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to decode comics"})
		}

		return c.JSON(fiber.Map{"comics": comics})
	})

	// Delete user
	admin.Delete("/users/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err = services.UsersCollection.DeleteOne(ctx, bson.M{"_id": objID})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to delete user"})
		}

		return c.JSON(fiber.Map{"message": "User deleted"})
	})

	// Delete video
	admin.Delete("/videos/:id", videoController.DeleteVideo)

	// Delete comic
	admin.Delete("/comics/:id", comicController.DeleteComic)

	// ============ SYNC ROUTES (Protected) ============
	// Sync Cloudinary/Mega content với database
	sync := api.Group("/sync", middleware.AuthMiddleware())

	// ========== SYNC VIDEOS (Cloudinary + Mega) ==========
	sync.Post("/videos", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(string)
		userObjID, err := primitive.ObjectIDFromHex(userID)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
		}

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		// Initialize services
		cloudinarySvc, cloudinaryErr := services.NewCloudinaryService()
		megaSvc, megaErr := services.NewMegaService()

		var syncedVideos int
		var syncErrors []string

		// Lấy videos từ Cloudinary
		var cloudVideos []services.CloudinaryResource
		if cloudinaryErr == nil {
			_, cloudVideos, err = cloudinarySvc.ListAllResources(ctx)
			if err != nil {
				syncErrors = append(syncErrors, "Cloudinary list resources error: "+err.Error())
			}
		} else {
			syncErrors = append(syncErrors, "Cloudinary connection error: "+cloudinaryErr.Error())
		}

		// Lấy videos từ Mega
		var megaVideos []services.MegaFileInfo
		if megaErr == nil {
			megaVideos, err = megaSvc.ListAllVideos()
			if err != nil {
				syncErrors = append(syncErrors, "Mega list videos error: "+err.Error())
			}
		} else {
			syncErrors = append(syncErrors, "Mega connection error (optional): "+megaErr.Error())
		}

		// Lấy existing video URLs từ DB
		existingVideosCursor, _ := services.VideosCollection.Find(ctx, bson.M{})
		existingVideoURLs := make(map[string]bool)
		existingVideoPublicIDs := make(map[string]bool)
		existingMegaHashes := make(map[string]bool)
		var existingVideos []bson.M
		existingVideosCursor.All(ctx, &existingVideos)
		for _, v := range existingVideos {
			if url, ok := v["video_url"].(string); ok && url != "" {
				existingVideoURLs[url] = true
			}
			if pid, ok := v["cloudinary_public_id"].(string); ok && pid != "" {
				existingVideoPublicIDs[pid] = true
			}
			if hash, ok := v["mega_hash"].(string); ok && hash != "" {
				existingMegaHashes[hash] = true
			}
		}

		// Insert videos từ Cloudinary
		for _, vid := range cloudVideos {
			if existingVideoURLs[vid.SecureURL] || existingVideoURLs[vid.URL] || existingVideoPublicIDs[vid.PublicID] {
				continue
			}

			parts := strings.Split(vid.PublicID, "/")
			title := parts[len(parts)-1]
			if len(parts) >= 2 {
				title = parts[len(parts)-2]
				if title == "videos" && len(parts) >= 1 {
					title = parts[len(parts)-1]
				}
			}
			title = strings.TrimSuffix(title, "."+vid.Format)

			newVideo := bson.M{
				"_id":                  primitive.NewObjectID(),
				"title":                title,
				"description":          "Đã đồng bộ từ Cloudinary",
				"thumbnail":            cloudinarySvc.GenerateThumbnail(vid.PublicID),
				"video_url":            vid.SecureURL,
				"cloudinary_public_id": vid.PublicID,
				"storage_type":         "cloudinary",
				"duration":             0,
				"duration_type":        "short",
				"quality":              "1080p",
				"file_size":            vid.Bytes,
				"tags":                 []string{},
				"genres":               []string{"Chưa phân loại"},
				"views":                int64(0),
				"likes":                int64(0),
				"dislikes":             int64(0),
				"status":               "ready",
				"uploaded_by":          userObjID,
				"created_at":           time.Now(),
				"updated_at":           time.Now(),
			}

			_, insertErr := services.VideosCollection.InsertOne(ctx, newVideo)
			if insertErr != nil {
				syncErrors = append(syncErrors, "Insert Cloudinary video error: "+insertErr.Error())
			} else {
				syncedVideos++
				existingVideoURLs[vid.SecureURL] = true
				existingVideoPublicIDs[vid.PublicID] = true
			}
		}

		// Insert videos từ Mega
		for _, vid := range megaVideos {
			if existingMegaHashes[vid.Hash] || existingVideoURLs[vid.PublicURL] {
				continue
			}

			title := strings.TrimSuffix(vid.Name, filepath.Ext(vid.Name))
			thumbnail := "https://placehold.co/640x360/1a1a2e/ffffff?text=" + strings.ReplaceAll(title, " ", "+")

			newVideo := bson.M{
				"_id":           primitive.NewObjectID(),
				"title":         title,
				"description":   "Đã đồng bộ từ Mega",
				"thumbnail":     thumbnail,
				"video_url":     vid.PublicURL,
				"mega_hash":     vid.Hash,
				"storage_type":  "mega",
				"duration":      0,
				"duration_type": "long",
				"quality":       "1080p",
				"file_size":     vid.Size,
				"tags":          []string{},
				"genres":        []string{"Chưa phân loại"},
				"views":         int64(0),
				"likes":         int64(0),
				"dislikes":      int64(0),
				"status":        "ready",
				"uploaded_by":   userObjID,
				"created_at":    time.Now(),
				"updated_at":    time.Now(),
			}

			_, insertErr := services.VideosCollection.InsertOne(ctx, newVideo)
			if insertErr != nil {
				syncErrors = append(syncErrors, "Insert Mega video error: "+insertErr.Error())
			} else {
				syncedVideos++
				existingMegaHashes[vid.Hash] = true
				existingVideoURLs[vid.PublicURL] = true
			}
		}

		result := fiber.Map{
			"message":           "Đồng bộ videos thành công",
			"synced_videos":     syncedVideos,
			"cloudinary_videos": len(cloudVideos),
			"mega_videos":       len(megaVideos),
		}
		if len(syncErrors) > 0 {
			result["errors"] = syncErrors
		}

		return c.JSON(result)
	})

	// ========== SYNC COMICS (Cloudinary) ==========
	sync.Post("/comics", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(string)
		userObjID, err := primitive.ObjectIDFromHex(userID)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
		}

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		// Initialize Cloudinary
		cloudinarySvc, cloudinaryErr := services.NewCloudinaryService()
		if cloudinaryErr != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Cloudinary connection error: " + cloudinaryErr.Error()})
		}

		var syncedComics int
		var syncErrors []string
		var debugInfo []string

		// List all root folders để debug
		folders, foldersErr := cloudinarySvc.ListAllFolders(ctx)
		if foldersErr != nil {
			debugInfo = append(debugInfo, "Cannot list folders: "+foldersErr.Error())
		} else {
			debugInfo = append(debugInfo, fmt.Sprintf("Root folders in Cloudinary: %v", folders))
		}

		// ============ CÁCH MỚI: List subfolders trong comics ============
		comicGroups := make(map[string][]services.CloudinaryResource)
		comicNames := make(map[string]string)

		// List subfolders trong "comics" folder
		comicSubfolders, subErr := cloudinarySvc.ListSubFolders(ctx, "comics")
		if subErr != nil {
			debugInfo = append(debugInfo, "ListSubFolders(comics) error: "+subErr.Error())
		} else {
			debugInfo = append(debugInfo, fmt.Sprintf("Comic subfolders: %v", comicSubfolders))

			// Với mỗi subfolder, lấy images
			for _, subfolderPath := range comicSubfolders {
				// subfolderPath = "comics/ten-truyen"
				images, imgErr := cloudinarySvc.ListImagesInFolder(ctx, subfolderPath)
				if imgErr != nil {
					debugInfo = append(debugInfo, fmt.Sprintf("Error listing images in %s: %v", subfolderPath, imgErr))
					continue
				}

				debugInfo = append(debugInfo, fmt.Sprintf("Folder '%s': %d images", subfolderPath, len(images)))

				if len(images) > 0 {
					// Lấy tên truyện từ path
					parts := strings.Split(subfolderPath, "/")
					folderName := parts[len(parts)-1]

					comicGroups[subfolderPath] = images
					comicNames[subfolderPath] = folderName
				}
			}
		}

		// Cũng thử list subfolders cho các tên folder khác
		for _, rootName := range []string{"comic", "truyen", "manga"} {
			subfolders, subErr := cloudinarySvc.ListSubFolders(ctx, rootName)
			if subErr == nil && len(subfolders) > 0 {
				debugInfo = append(debugInfo, fmt.Sprintf("%s subfolders: %v", rootName, subfolders))
				for _, subfolderPath := range subfolders {
					if _, exists := comicGroups[subfolderPath]; exists {
						continue
					}
					images, imgErr := cloudinarySvc.ListImagesInFolder(ctx, subfolderPath)
					if imgErr == nil && len(images) > 0 {
						parts := strings.Split(subfolderPath, "/")
						folderName := parts[len(parts)-1]
						comicGroups[subfolderPath] = images
						comicNames[subfolderPath] = folderName
						debugInfo = append(debugInfo, fmt.Sprintf("Folder '%s': %d images", subfolderPath, len(images)))
					}
				}
			}
		}

		debugInfo = append(debugInfo, fmt.Sprintf("Comic groups found: %d", len(comicGroups)))
		for key, imgs := range comicGroups {
			debugInfo = append(debugInfo, fmt.Sprintf("Group '%s': %d images", key, len(imgs)))
		}

		// Lấy existing comics từ DB
		existingComicsCursor, _ := services.ComicsCollection.Find(ctx, bson.M{})
		existingCoverURLs := make(map[string]bool)
		existingTitles := make(map[string]bool)
		existingFolderPaths := make(map[string]bool)
		var existingComics []bson.M
		existingComicsCursor.All(ctx, &existingComics)
		for _, c := range existingComics {
			if cover, ok := c["cover_image"].(string); ok && cover != "" {
				existingCoverURLs[cover] = true
			}
			if title, ok := c["title"].(string); ok && title != "" {
				existingTitles[strings.ToLower(title)] = true
			}
			if fp, ok := c["folder_path"].(string); ok && fp != "" {
				existingFolderPaths[fp] = true
			}
		}

		// Tạo comic từ mỗi folder
		for folderKey, images := range comicGroups {
			if len(images) == 0 {
				continue
			}

			// Skip nếu folder path đã tồn tại
			if existingFolderPaths[folderKey] {
				debugInfo = append(debugInfo, fmt.Sprintf("Skip folder '%s': already exists by folder_path", folderKey))
				continue
			}

			// Sort images theo PublicID
			sort.Slice(images, func(i, j int) bool {
				return images[i].PublicID < images[j].PublicID
			})

			coverURL := images[0].SecureURL

			// Format title
			title := comicNames[folderKey]
			title = strings.ReplaceAll(title, "_", " ")
			title = strings.ReplaceAll(title, "-", " ")

			// Skip nếu cover hoặc title đã tồn tại
			if existingCoverURLs[coverURL] {
				debugInfo = append(debugInfo, fmt.Sprintf("Skip folder '%s': cover already exists", folderKey))
				continue
			}
			if existingTitles[strings.ToLower(title)] {
				debugInfo = append(debugInfo, fmt.Sprintf("Skip folder '%s': title '%s' already exists", folderKey, title))
				continue
			}

			// Tạo chapter images
			chapterImages := make([]bson.M, 0, len(images))
			for i, img := range images {
				chapterImages = append(chapterImages, bson.M{
					"page":      i + 1,
					"url":       img.SecureURL,
					"public_id": img.PublicID,
					"width":     img.Width,
					"height":    img.Height,
				})
			}

			newComic := bson.M{
				"_id":         primitive.NewObjectID(),
				"title":       title,
				"description": fmt.Sprintf("Đã đồng bộ từ Cloudinary (%d trang)", len(images)),
				"author":      "Chưa rõ",
				"cover_image": coverURL,
				"folder_path": folderKey,
				"tags":        []string{},
				"genres":      []string{"Chưa phân loại"},
				"status":      "ongoing",
				"chapters": []bson.M{
					{
						"_id":         primitive.NewObjectID(),
						"number":      1,
						"title":       "Chapter 1",
						"images":      chapterImages,
						"views":       int64(0),
						"uploaded_at": time.Now(),
					},
				},
				"views":        int64(0),
				"likes":        int64(0),
				"rating":       float64(0),
				"rating_count": int64(0),
				"uploaded_by":  userObjID,
				"created_at":   time.Now(),
				"updated_at":   time.Now(),
			}

			_, insertErr := services.ComicsCollection.InsertOne(ctx, newComic)
			if insertErr != nil {
				syncErrors = append(syncErrors, "Insert comic '"+title+"' error: "+insertErr.Error())
				debugInfo = append(debugInfo, fmt.Sprintf("Failed to insert comic '%s': %v", title, insertErr))
			} else {
				syncedComics++
				existingCoverURLs[coverURL] = true
				existingTitles[strings.ToLower(title)] = true
				existingFolderPaths[folderKey] = true
				debugInfo = append(debugInfo, fmt.Sprintf("Successfully created comic '%s' with %d pages", title, len(images)))
			}
		}

		// Return result với debug info
		var folderNames []string
		totalImages := 0
		for key, imgs := range comicGroups {
			folderNames = append(folderNames, key)
			totalImages += len(imgs)
		}

		result := fiber.Map{
			"message":          "Đồng bộ truyện thành công",
			"synced_comics":    syncedComics,
			"total_images":     totalImages,
			"comic_folders":    len(comicGroups),
			"detected_folders": folderNames,
			"debug":            debugInfo,
		}
		if len(syncErrors) > 0 {
			result["errors"] = syncErrors
		}

		return c.JSON(result)
	})
}
