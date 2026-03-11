package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"media-backend/models"
	"media-backend/services"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ============ CONFIGURATION ============

var (
	// Folder chứa các subfolder comic (mỗi subfolder = 1 comic)
	ComicsFolder = "./comics"
)

// Supported image formats
var supportedImageFormats = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
}

// ============ COMIC INFO ============

// ComicFolder holds information about a comic folder to upload
type ComicFolder struct {
	Path       string   // Full path to folder
	Name       string   // Folder name (will be used as comic title)
	ImagePaths []string // List of image paths in the folder
	ImageCount int
	Uploaded   bool
	Error      error
	MongoID    string // MongoDB document ID after saving
	CoverURL   string // Cloudinary URL of cover image
}

// ============ BATCH UPLOADER ============

// BatchComicUploader handles batch comic uploads
type BatchComicUploader struct {
	cloudinaryService *services.CloudinaryService
	comics            []*ComicFolder
	logger            *log.Logger

	// Stats
	totalComics   int
	uploadedCount int
	failedCount   int
	skippedCount  int
	totalImages   int
	startTime     time.Time
}

// NewBatchComicUploader creates a new batch uploader instance
func NewBatchComicUploader() (*BatchComicUploader, error) {
	// Create Cloudinary service
	cloudinary, err := services.NewCloudinaryService()
	if err != nil {
		return nil, fmt.Errorf("failed to create cloudinary service: %w", err)
	}

	// Create logger
	logger := log.New(os.Stdout, "", 0)

	return &BatchComicUploader{
		cloudinaryService: cloudinary,
		logger:            logger,
		comics:            make([]*ComicFolder, 0),
	}, nil
}

// ============ LOGGING HELPERS ============

func (b *BatchComicUploader) logInfo(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	b.logger.Printf("[%s] [INFO] %s", timestamp, fmt.Sprintf(format, args...))
}

func (b *BatchComicUploader) logSuccess(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	b.logger.Printf("[%s] [SUCCESS] %s", timestamp, fmt.Sprintf(format, args...))
}

func (b *BatchComicUploader) logWarning(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	b.logger.Printf("[%s] [WARNING] %s", timestamp, fmt.Sprintf(format, args...))
}

func (b *BatchComicUploader) logError(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	b.logger.Printf("[%s] [ERROR] %s", timestamp, fmt.Sprintf(format, args...))
}

// ============ SCANNING ============

// ScanFolder scans a folder for comic subfolders
func (b *BatchComicUploader) ScanFolder(rootFolder string) error {
	b.logInfo("Scanning folder: %s", rootFolder)

	// Check folder exists
	info, err := os.Stat(rootFolder)
	if err != nil {
		return fmt.Errorf("folder not found: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", rootFolder)
	}

	// Walk through subfolders
	entries, err := os.ReadDir(rootFolder)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue // Skip files, only process folders
		}

		comicPath := filepath.Join(rootFolder, entry.Name())
		comic := &ComicFolder{
			Path:       comicPath,
			Name:       entry.Name(),
			ImagePaths: make([]string, 0),
		}

		// Scan images in comic folder
		err := filepath.WalkDir(comicPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil // Skip subdirectories
			}

			ext := strings.ToLower(filepath.Ext(path))
			if supportedImageFormats[ext] {
				comic.ImagePaths = append(comic.ImagePaths, path)
			}
			return nil
		})
		if err != nil {
			b.logWarning("Error scanning comic folder %s: %v", entry.Name(), err)
			continue
		}

		// Sort images by filename (natural order)
		sort.Strings(comic.ImagePaths)
		comic.ImageCount = len(comic.ImagePaths)

		if comic.ImageCount > 0 {
			b.comics = append(b.comics, comic)
			b.logInfo("Found comic: '%s' with %d images", comic.Name, comic.ImageCount)
		} else {
			b.logWarning("Skipping empty folder: %s", entry.Name())
		}
	}

	// Sort comics by name
	sort.Slice(b.comics, func(i, j int) bool {
		return b.comics[i].Name < b.comics[j].Name
	})

	b.totalComics = len(b.comics)
	b.logInfo("Found %d comics to upload", b.totalComics)

	// Count total images
	for _, comic := range b.comics {
		b.totalImages += comic.ImageCount
	}
	b.logInfo("Total images to upload: %d", b.totalImages)

	return nil
}

// ============ UPLOAD ============

// UploadAll uploads all comics
func (b *BatchComicUploader) UploadAll(ctx context.Context) error {
	b.startTime = time.Now()
	b.logInfo("Starting batch upload of %d comics (%d images total)", b.totalComics, b.totalImages)
	b.logInfo("=" + strings.Repeat("=", 60))

	// Connect to database
	if err := services.ConnectDB(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	b.logInfo("Connected to MongoDB")

	for i, comic := range b.comics {
		select {
		case <-ctx.Done():
			b.logWarning("Upload cancelled by user")
			return ctx.Err()
		default:
		}

		b.logInfo("[%d/%d] Uploading comic: '%s' (%d images)...",
			i+1, b.totalComics, comic.Name, comic.ImageCount)

		err := b.uploadComic(ctx, comic)
		if err != nil {
			comic.Error = err
			b.failedCount++
			b.logError("Failed to upload '%s': %v", comic.Name, err)
			continue
		}

		comic.Uploaded = true
		b.uploadedCount++
		b.logSuccess("Uploaded '%s' successfully (ID: %s, Cover: %s)",
			comic.Name, comic.MongoID, comic.CoverURL)

		// Small delay between comics to avoid rate limits
		if i < b.totalComics-1 {
			time.Sleep(2 * time.Second)
		}
	}

	b.printSummary()
	return nil
}

// uploadComic uploads a single comic
func (b *BatchComicUploader) uploadComic(ctx context.Context, comic *ComicFolder) error {
	comicID := primitive.NewObjectID()
	folder := fmt.Sprintf("comics/%s", comicID.Hex())

	// Upload images to Cloudinary
	var uploadedImages []models.ComicImage
	for i, imagePath := range comic.ImagePaths {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		result, err := b.cloudinaryService.UploadImageFromPath(ctx, imagePath, folder)
		if err != nil {
			return fmt.Errorf("failed to upload image %s: %w", filepath.Base(imagePath), err)
		}

		uploadedImages = append(uploadedImages, models.ComicImage{
			Page:     i + 1,
			URL:      result.SecureURL,
			PublicID: result.PublicID,
			Width:    result.Width,
			Height:   result.Height,
		})

		// Progress log every 10 images
		if (i+1)%10 == 0 || i == len(comic.ImagePaths)-1 {
			b.logInfo("  Uploaded %d/%d images", i+1, len(comic.ImagePaths))
		}

		// Small delay between images
		time.Sleep(200 * time.Millisecond)
	}

	// Create chapter
	now := time.Now()
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

	// Create comic document
	comicDoc := models.Comic{
		ID:          comicID,
		Title:       comic.Name,
		Description: "Truyện hành động",
		Author:      "",
		CoverImage:  coverImage,
		Tags:        []string{"action"},
		Genres:      []string{"Action"},
		Status:      "ongoing",
		Chapters:    []models.Chapter{chapter},
		Views:       0,
		Likes:       0,
		UploadedBy:  primitive.NilObjectID, // System upload
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Insert into database
	dbCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := services.ComicsCollection.InsertOne(dbCtx, comicDoc)
	if err != nil {
		return fmt.Errorf("failed to insert comic: %w", err)
	}

	comic.MongoID = comicID.Hex()
	comic.CoverURL = coverImage
	return nil
}

// ============ SUMMARY ============

func (b *BatchComicUploader) printSummary() {
	elapsed := time.Since(b.startTime)

	b.logInfo("=" + strings.Repeat("=", 60))
	b.logInfo("BATCH UPLOAD COMPLETED")
	b.logInfo("=" + strings.Repeat("=", 60))
	b.logInfo("Total comics:   %d", b.totalComics)
	b.logInfo("Uploaded:       %d", b.uploadedCount)
	b.logInfo("Failed:         %d", b.failedCount)
	b.logInfo("Skipped:        %d", b.skippedCount)
	b.logInfo("Total images:   %d", b.totalImages)
	b.logInfo("Time elapsed:   %s", elapsed.Round(time.Second))
	b.logInfo("=" + strings.Repeat("=", 60))

	// List failed comics
	if b.failedCount > 0 {
		b.logError("Failed comics:")
		for _, comic := range b.comics {
			if comic.Error != nil {
				b.logError("  - %s: %v", comic.Name, comic.Error)
			}
		}
	}
}

// ============ MAIN ============

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║           BATCH COMIC UPLOADER - CLOUDINARY EDITION           ║")
	fmt.Println("║      Upload multiple comics from local folders                ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Check command line arguments
	if len(os.Args) > 1 {
		ComicsFolder = os.Args[1]
	}

	fmt.Printf("Comics folder: %s\n", ComicsFolder)
	fmt.Println()

	// Create uploader
	uploader, err := NewBatchComicUploader()
	if err != nil {
		log.Fatalf("Failed to initialize uploader: %v", err)
	}

	// Scan folder
	if err := uploader.ScanFolder(ComicsFolder); err != nil {
		log.Fatalf("Failed to scan folder: %v", err)
	}

	if uploader.totalComics == 0 {
		fmt.Println("No comics found to upload.")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  1. Create a folder structure like:")
		fmt.Println("     comics/")
		fmt.Println("       ├── Naruto/")
		fmt.Println("       │   ├── 001.jpg")
		fmt.Println("       │   ├── 002.jpg")
		fmt.Println("       │   └── ...")
		fmt.Println("       ├── One Piece/")
		fmt.Println("       │   ├── 001.jpg")
		fmt.Println("       │   └── ...")
		fmt.Println("       └── ...")
		fmt.Println()
		fmt.Println("  2. Run: go run ./cmd/batch_upload_comics ./comics")
		fmt.Println()
		fmt.Println("Each subfolder name becomes the comic title.")
		fmt.Println("Supported image formats: jpg, jpeg, png, gif, webp")
		return
	}

	// Print summary and confirm
	fmt.Println()
	fmt.Println("Comics to upload:")
	for i, comic := range uploader.comics {
		fmt.Printf("  %d. %s (%d images)\n", i+1, comic.Name, comic.ImageCount)
	}
	fmt.Println()

	fmt.Print("Press Enter to start uploading, or Ctrl+C to cancel...")
	fmt.Scanln()
	fmt.Println()

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\nInterrupt received, cancelling...")
		cancel()
	}()

	// Start upload
	if err := uploader.UploadAll(ctx); err != nil {
		if err != context.Canceled {
			log.Fatalf("Upload failed: %v", err)
		}
	}

	fmt.Println()
	fmt.Println("Done!")
}
