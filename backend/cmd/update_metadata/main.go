package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"media-backend/models"
	"media-backend/services"
	"media-backend/services/telegram"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║     VIDEO METADATA UPDATER - Telegram Videos                ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: Could not load .env file")
	}

	// Check FFprobe
	if !telegram.IsFFprobeAvailable() {
		log.Fatal("❌ FFprobe not found! Please install FFmpeg first.")
	}
	fmt.Println("✅ FFprobe available")

	// Connect to MongoDB
	ctx := context.Background()
	if err := services.ConnectDB(); err != nil {
		log.Fatalf("❌ Failed to connect to MongoDB: %v", err)
	}
	fmt.Println("✅ Connected to MongoDB")

	// Initialize Telegram service
	telegramService, err := telegram.NewTelegramService()
	if err != nil {
		log.Fatalf("❌ Failed to create Telegram service: %v", err)
	}

	// Find videos needing metadata update first
	filter := bson.M{
		"storage_provider": "telegram",
		"$or": []bson.M{
			{"duration": 0},
			{"duration": bson.M{"$exists": false}},
			{"thumbnail": ""},
			{"thumbnail": bson.M{"$exists": false}},
		},
	}

	collection := services.VideosCollection
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		log.Fatalf("❌ Failed to query videos: %v", err)
	}

	var videos []models.Video
	if err := cursor.All(ctx, &videos); err != nil {
		cursor.Close(ctx)
		log.Fatalf("❌ Failed to decode videos: %v", err)
	}
	cursor.Close(ctx)

	if len(videos) == 0 {
		fmt.Println("\n✅ No videos need metadata update!")
		return
	}

	fmt.Printf("\n📹 Found %d videos needing metadata update\n\n", len(videos))

	// Create temp directory
	tempDir := filepath.Join(os.TempDir(), "media_metadata_update")
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	// Run with Telegram connection
	err = telegramService.RunWithCallback(ctx, func(ctx context.Context) error {
		fmt.Println("✅ Telegram connected")

		// Process each video
		for i, video := range videos {
			processVideo(ctx, telegramService, collection, video, i+1, len(videos), tempDir)
		}
		return nil
	})

	if err != nil {
		log.Printf("Error during processing: %v", err)
	}

	fmt.Println("\n╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    UPDATE COMPLETE!                         ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
}

func processVideo(ctx context.Context, ts *telegram.TelegramService, collection *mongo.Collection, video models.Video, num, total int, tempDir string) {
	fmt.Printf("[%d/%d] Processing: %s\n", num, total, video.Title)

	// Download video from Telegram
	tempFile := filepath.Join(tempDir, fmt.Sprintf("video_%s%s", video.ID.Hex(), getExtFromMime(video.MimeType)))

	fmt.Printf("  ⏳ Downloading from Telegram (message_id=%d)...\n", video.TelegramMessageID)

	err := downloadFromTelegram(ctx, ts, video.TelegramMessageID, tempFile)
	if err != nil {
		fmt.Printf("  ❌ Failed to download: %v\n\n", err)
		return
	}
	fmt.Println("  ✅ Downloaded")

	// Extract metadata
	fmt.Println("  ⏳ Extracting metadata...")
	metadata, err := telegram.ExtractVideoMetadata(tempFile)
	if err != nil {
		fmt.Printf("  ❌ Failed to extract metadata: %v\n\n", err)
		os.Remove(tempFile)
		return
	}
	fmt.Printf("  ✅ Duration: %ds, Resolution: %dx%d\n", metadata.Duration, metadata.Width, metadata.Height)

	// Generate thumbnail
	thumbnailPath := tempFile + "_thumb.jpg"
	fmt.Println("  ⏳ Generating thumbnail...")
	if err := generateThumbnail(tempFile, thumbnailPath, metadata.Duration); err != nil {
		fmt.Printf("  ⚠️ Thumbnail generation failed: %v\n", err)
	}

	// Upload thumbnail to Cloudinary if generated
	var thumbnailURL string
	if _, err := os.Stat(thumbnailPath); err == nil {
		fmt.Println("  ⏳ Uploading thumbnail to Cloudinary...")
		thumbnailURL, err = uploadThumbnailToCloudinary(ctx, thumbnailPath, video.ID.Hex())
		if err != nil {
			fmt.Printf("  ⚠️ Thumbnail upload failed: %v\n", err)
		} else {
			fmt.Printf("  ✅ Thumbnail uploaded: %s\n", thumbnailURL)
		}
		os.Remove(thumbnailPath)
	}

	// Update MongoDB
	update := bson.M{
		"$set": bson.M{
			"duration":   metadata.Duration,
			"width":      metadata.Width,
			"height":     metadata.Height,
			"updated_at": time.Now(),
		},
	}
	if thumbnailURL != "" {
		update["$set"].(bson.M)["thumbnail"] = thumbnailURL
	}

	// Determine quality
	quality := determineQuality(metadata.Width, metadata.Height)
	update["$set"].(bson.M)["quality"] = quality

	_, err = collection.UpdateByID(ctx, video.ID, update)
	if err != nil {
		fmt.Printf("  ❌ Failed to update MongoDB: %v\n\n", err)
	} else {
		fmt.Printf("  ✅ Updated in MongoDB (quality: %s)\n\n", quality)
	}

	// Clean up
	os.Remove(tempFile)

	// Small delay between videos
	time.Sleep(2 * time.Second)
}

func downloadFromTelegram(ctx context.Context, ts *telegram.TelegramService, messageID int, destPath string) error {
	// Download video directly - must be called within RunWithCallback
	data, err := ts.DownloadVideoDirect(ctx, messageID)
	if err != nil {
		return err
	}

	return os.WriteFile(destPath, data, 0644)
}

func generateThumbnail(videoPath, thumbPath string, duration int) error {
	// Seek to 10% of video or 5 seconds, whichever is smaller
	seekTime := duration / 10
	if seekTime > 5 {
		seekTime = 5
	}
	if seekTime < 1 {
		seekTime = 1
	}

	cmd := exec.Command("ffmpeg",
		"-y",
		"-ss", fmt.Sprintf("%d", seekTime),
		"-i", videoPath,
		"-vframes", "1",
		"-vf", "scale=640:-1",
		"-q:v", "2",
		thumbPath,
	)

	return cmd.Run()
}

func uploadThumbnailToCloudinary(ctx context.Context, thumbPath, videoID string) (string, error) {
	// Check if file exists and has content
	info, err := os.Stat(thumbPath)
	if err != nil {
		return "", fmt.Errorf("thumbnail file not found: %w", err)
	}
	if info.Size() == 0 {
		return "", fmt.Errorf("thumbnail file is empty")
	}
	fmt.Printf("    [Debug] Thumbnail file size: %d bytes\n", info.Size())

	cloudinaryService, err := services.NewCloudinaryService()
	if err != nil {
		return "", fmt.Errorf("failed to create cloudinary service: %w", err)
	}

	result, err := cloudinaryService.UploadImageFromPath(ctx, thumbPath, "thumbnails")
	if err != nil {
		return "", fmt.Errorf("upload failed: %w", err)
	}

	fmt.Printf("    [Debug] Cloudinary result - URL: %s, SecureURL: %s, PublicID: %s\n",
		result.URL, result.SecureURL, result.PublicID)

	if result.SecureURL == "" {
		return "", fmt.Errorf("cloudinary returned empty secure URL")
	}

	return result.SecureURL, nil
}

func getExtFromMime(mimeType string) string {
	switch mimeType {
	case "video/mp4":
		return ".mp4"
	case "video/x-matroska":
		return ".mkv"
	case "video/quicktime":
		return ".mov"
	case "video/webm":
		return ".webm"
	default:
		return ".mp4"
	}
}

func determineQuality(width, height int) string {
	maxDim := width
	if height > width {
		maxDim = height
	}

	switch {
	case maxDim >= 2160:
		return "4K"
	case maxDim >= 1440:
		return "1440p"
	case maxDim >= 1080:
		return "1080p"
	case maxDim >= 720:
		return "720p"
	case maxDim >= 480:
		return "480p"
	case maxDim >= 360:
		return "360p"
	default:
		return "240p"
	}
}

// Placeholder for video ID conversion
func init() {
	_ = primitive.NewObjectID()
}
