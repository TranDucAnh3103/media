package main

import (
	"context"
	"errors"
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
	"media-backend/services/telegram"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ============ CONFIGURATION ============

var (
	// Folder containing videos to upload (can be overridden via command line)
	VideoFolder = "./videos"
)

const (
	// Supported video formats
	SupportedFormats = ".mp4,.mkv,.mov,.avi,.webm"

	// Max retries per video
	MaxRetries = 3

	// Size thresholds in bytes
	Size200MB  = 200 * 1024 * 1024  // 200 MB
	Size800MB  = 800 * 1024 * 1024  // 800 MB
	Size1200MB = 1200 * 1024 * 1024 // 1.2 GB
	Size2000MB = 2000 * 1024 * 1024 // 2 GB
)

// Delay durations based on file size
var delayRules = []struct {
	MaxSize int64
	Delay   time.Duration
}{
	{Size200MB, 1 * time.Minute},  // Video < 200 MB → 1 minute delay
	{Size800MB, 2 * time.Minute},  // Video 200 MB – 800 MB → 2 minutes delay
	{Size1200MB, 4 * time.Minute}, // Video 800 MB – 1.2 GB → 4 minutes delay
	{Size2000MB, 6 * time.Minute}, // Video 1.2 GB – 2 GB → 6 minutes delay
}

// ============ VIDEO INFO ============

// VideoFile holds information about a video file to upload
type VideoFile struct {
	Path         string
	Name         string
	Size         int64
	SizeStr      string
	Uploaded     bool
	Error        error
	Retries      int
	MessageID    int    // Telegram message ID after upload
	FileID       string // Telegram file ID after upload
	MongoID      string // MongoDB document ID after saving
	ThumbnailURL string // Cloudinary thumbnail URL
	Duration     int    // Video duration in seconds
	Width        int    // Video width
	Height       int    // Video height
}

// ============ BATCH UPLOADER ============

// BatchUploader handles batch video uploads
type BatchUploader struct {
	telegramService   *telegram.TelegramService
	cloudinaryService *services.CloudinaryService
	videos            []*VideoFile
	logger            *log.Logger

	// Stats
	totalVideos   int
	uploadedCount int
	failedCount   int
	skippedCount  int
	totalBytesUp  int64
	startTime     time.Time

	// FloodWait tracking
	floodWaitEvents int
	totalWaitTime   time.Duration
}

// NewBatchUploader creates a new batch uploader instance
func NewBatchUploader() (*BatchUploader, error) {
	// Create Telegram service
	service, err := telegram.NewTelegramService()
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram service: %w", err)
	}

	// Create Cloudinary service for thumbnails
	cloudinary, err := services.NewCloudinaryService()
	if err != nil {
		fmt.Printf("Warning: Cloudinary not available, thumbnails will be skipped: %v\n", err)
	}

	// Create logger with timestamp
	logger := log.New(os.Stdout, "", 0)

	return &BatchUploader{
		telegramService:   service,
		cloudinaryService: cloudinary,
		logger:            logger,
		videos:            make([]*VideoFile, 0),
	}, nil
}

// ============ LOGGING HELPERS ============

func (b *BatchUploader) logInfo(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	b.logger.Printf("[%s] [INFO] %s", timestamp, fmt.Sprintf(format, args...))
}

func (b *BatchUploader) logSuccess(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	b.logger.Printf("[%s] [SUCCESS] %s", timestamp, fmt.Sprintf(format, args...))
}

func (b *BatchUploader) logWarning(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	b.logger.Printf("[%s] [WARNING] %s", timestamp, fmt.Sprintf(format, args...))
}

func (b *BatchUploader) logError(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	b.logger.Printf("[%s] [ERROR] %s", timestamp, fmt.Sprintf(format, args...))
}

func (b *BatchUploader) logFloodWait(waitSeconds int) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	b.logger.Printf("[%s] [FLOODWAIT] Telegram requires waiting %d seconds", timestamp, waitSeconds)
}

func (b *BatchUploader) logDelay(duration time.Duration, reason string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	b.logger.Printf("[%s] [DELAY] Waiting %s (%s)", timestamp, duration, reason)
}

// ============ UTILITY FUNCTIONS ============

// formatSize converts bytes to human-readable format
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// isSupportedFormat checks if the file extension is supported
func isSupportedFormat(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	supported := strings.Split(SupportedFormats, ",")
	for _, s := range supported {
		if ext == s {
			return true
		}
	}
	return false
}

// getDelayForSize returns the delay duration based on file size
func getDelayForSize(size int64) time.Duration {
	for _, rule := range delayRules {
		if size < rule.MaxSize {
			return rule.Delay
		}
	}
	// For files >= 2GB, use 6 minutes delay
	return 6 * time.Minute
}

// ============ FOLDER SCANNING ============

// ScanFolder scans the video folder and collects video files
func (b *BatchUploader) ScanFolder() error {
	b.logInfo("Scanning folder: %s", VideoFolder)

	// Check if folder exists
	info, err := os.Stat(VideoFolder)
	if os.IsNotExist(err) {
		return fmt.Errorf("video folder does not exist: %s", VideoFolder)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", VideoFolder)
	}

	// Walk through folder
	err = filepath.WalkDir(VideoFolder, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Check if file is a supported video format
		if !isSupportedFormat(d.Name()) {
			b.logInfo("Skipping unsupported file: %s", d.Name())
			return nil
		}

		// Get file info
		fileInfo, err := d.Info()
		if err != nil {
			b.logWarning("Failed to get file info for %s: %v", d.Name(), err)
			return nil
		}

		// Add to video list
		video := &VideoFile{
			Path:    path,
			Name:    d.Name(),
			Size:    fileInfo.Size(),
			SizeStr: formatSize(fileInfo.Size()),
		}
		b.videos = append(b.videos, video)

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan folder: %w", err)
	}

	// Sort by file size (smallest first)
	sort.Slice(b.videos, func(i, j int) bool {
		return b.videos[i].Size < b.videos[j].Size
	})

	b.totalVideos = len(b.videos)
	b.logInfo("Found %d video files", b.totalVideos)

	// Log video list
	for i, v := range b.videos {
		b.logInfo("  [%d] %s (%s)", i+1, v.Name, v.SizeStr)
	}

	return nil
}

// ============ UPLOAD LOGIC ============

// extractVideoMetadata extracts metadata and generates thumbnail
func (b *BatchUploader) extractVideoMetadata(ctx context.Context, video *VideoFile) error {
	b.logInfo("Extracting video metadata...")

	// Check ffprobe availability first
	if !telegram.IsFFprobeAvailable() {
		b.logWarning("FFprobe not found in PATH - metadata extraction will be limited")
		b.logWarning("Install FFmpeg to enable full metadata extraction: https://ffmpeg.org/download.html")
		return nil
	}

	// Extract metadata using ffprobe
	metadata, err := telegram.ExtractVideoMetadata(video.Path)
	if err != nil {
		b.logWarning("Failed to extract metadata: %v", err)
		return nil // Non-fatal, continue without metadata
	}

	video.Duration = metadata.Duration
	video.Width = metadata.Width
	video.Height = metadata.Height

	if metadata.Duration > 0 {
		minutes := metadata.Duration / 60
		seconds := metadata.Duration % 60
		b.logInfo("  Duration: %d:%02d (%d seconds)", minutes, seconds, metadata.Duration)
	} else {
		b.logWarning("  Duration: could not be extracted")
	}

	if metadata.Width > 0 && metadata.Height > 0 {
		b.logInfo("  Resolution: %dx%d (%s)", metadata.Width, metadata.Height, telegram.GetQualityFromResolution(metadata.Height))
	} else {
		b.logWarning("  Resolution: could not be extracted")
	}

	// Extract and upload thumbnail to Cloudinary
	if !telegram.IsFFmpegAvailable() {
		b.logWarning("FFmpeg not available - skipping thumbnail extraction")
		return nil
	}

	if b.cloudinaryService == nil {
		b.logWarning("Cloudinary not configured - skipping thumbnail upload")
		return nil
	}

	b.logInfo("Extracting thumbnail...")
	thumbnailPath, err := telegram.ExtractThumbnail(video.Path, os.TempDir())
	if err != nil {
		b.logWarning("Failed to extract thumbnail: %v", err)
		return nil
	}
	defer os.Remove(thumbnailPath) // Clean up temp file

	// Upload to Cloudinary
	result, err := b.cloudinaryService.UploadImageFromPath(ctx, thumbnailPath, "video_thumbnails")
	if err != nil {
		b.logWarning("Failed to upload thumbnail to Cloudinary: %v", err)
	} else {
		video.ThumbnailURL = result.SecureURL
		b.logSuccess("Thumbnail uploaded: %s", result.SecureURL)
	}

	return nil
}

// uploadSingleVideo uploads a single video with retry logic
func (b *BatchUploader) uploadSingleVideo(ctx context.Context, video *VideoFile) error {
	startTime := time.Now()

	b.logInfo("Starting upload: %s (%s)", video.Name, video.SizeStr)
	b.logInfo("Upload start time: %s", startTime.Format("2006-01-02 15:04:05"))

	// Extract metadata and thumbnail first
	if err := b.extractVideoMetadata(ctx, video); err != nil {
		b.logWarning("Metadata extraction failed: %v", err)
	}

	// Create upload request with metadata
	req := telegram.VideoUploadRequest{
		FilePath: video.Path,
		FileName: video.Name,
		Caption:  fmt.Sprintf("Batch upload: %s", video.Name),
		Duration: video.Duration,
		Width:    video.Width,
		Height:   video.Height,
		ProgressCb: func(progress telegram.UploadProgress) {
			if progress.Percent > 0 && int(progress.Percent)%20 == 0 {
				b.logInfo("  Progress: %.1f%% (%s / %s)",
					progress.Percent,
					formatSize(progress.BytesUploaded),
					formatSize(progress.TotalBytes))
			}
		},
	}

	// Upload with retry
	var lastErr error
	for attempt := 1; attempt <= MaxRetries; attempt++ {
		if attempt > 1 {
			b.logInfo("Retry attempt %d/%d for %s", attempt, MaxRetries, video.Name)
		}

		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Attempt upload
		result, err := b.telegramService.UploadVideoUnprotected(ctx, req)
		if err == nil {
			// Success
			duration := time.Since(startTime)
			b.logSuccess("Upload completed: %s", video.Name)
			b.logInfo("  Message ID: %d", result.MessageID)
			b.logInfo("  File ID: %s", result.FileID)
			b.logInfo("  Duration: %s", duration.Round(time.Second))
			b.logInfo("  Upload speed: %s/s", formatSize(int64(float64(video.Size)/duration.Seconds())))

			// Save to MongoDB
			video.MessageID = result.MessageID
			video.FileID = result.FileID

			mongoID, saveErr := b.saveVideoToDatabase(ctx, video, result)
			if saveErr != nil {
				b.logWarning("Failed to save to database: %v", saveErr)
			} else {
				video.MongoID = mongoID
				b.logSuccess("Saved to database: %s", mongoID)
			}

			video.Uploaded = true
			b.uploadedCount++
			b.totalBytesUp += video.Size
			return nil
		}

		lastErr = err
		video.Retries = attempt

		// Check if it's a FloodWait error
		waitSeconds := parseFloodWaitError(err)
		if waitSeconds > 0 {
			b.floodWaitEvents++
			b.logFloodWait(waitSeconds)

			waitDuration := time.Duration(waitSeconds) * time.Second
			b.totalWaitTime += waitDuration

			// Wait for FloodWait
			b.logInfo("Pausing upload for %s due to FloodWait", waitDuration)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitDuration):
				b.logInfo("Resuming after FloodWait")
			}
			continue
		}

		// For other errors, wait briefly before retry
		if attempt < MaxRetries {
			retryDelay := time.Duration(attempt*10) * time.Second
			b.logWarning("Upload failed: %v, retrying in %s", err, retryDelay)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
			}
		}
	}

	// All retries exhausted
	video.Error = lastErr
	b.failedCount++
	return fmt.Errorf("upload failed after %d retries: %w", MaxRetries, lastErr)
}

// parseFloodWaitError extracts wait seconds from FloodWait error
func parseFloodWaitError(err error) int {
	if err == nil {
		return 0
	}

	errStr := err.Error()

	// Common FloodWait patterns
	patterns := []string{
		"FLOOD_WAIT_",
		"FloodWait_",
		"flood_wait_",
		"retry after ",
		"retry_after:",
	}

	for _, pattern := range patterns {
		if idx := strings.Index(errStr, pattern); idx >= 0 {
			// Extract number after pattern
			numStr := ""
			for i := idx + len(pattern); i < len(errStr); i++ {
				if errStr[i] >= '0' && errStr[i] <= '9' {
					numStr += string(errStr[i])
				} else {
					break
				}
			}
			if numStr != "" {
				var seconds int
				fmt.Sscanf(numStr, "%d", &seconds)
				return seconds
			}
		}
	}

	// Check for HTTP 429
	if strings.Contains(errStr, "429") {
		return 60 // Default 60 seconds for 429
	}

	return 0
}

// saveVideoToDatabase saves video metadata to MongoDB after successful upload
func (b *BatchUploader) saveVideoToDatabase(ctx context.Context, video *VideoFile, result *telegram.VideoUploadResult) (string, error) {
	// Get channel ID from telegram service
	channelID := b.telegramService.GetChannelID()

	// Extract title from filename (remove extension)
	title := strings.TrimSuffix(video.Name, filepath.Ext(video.Name))

	// PRIORITY: FFprobe metadata (video.*) > Telegram response (result.*)
	// FFprobe is more accurate, especially for MKV and other container formats
	duration := video.Duration
	if duration == 0 && result.Duration > 0 {
		duration = result.Duration
	}
	width := video.Width
	if width == 0 && result.Width > 0 {
		width = result.Width
	}
	height := video.Height
	if height == 0 && result.Height > 0 {
		height = result.Height
	}

	// Determine duration type from final duration value
	durationType := ""
	if duration > 0 {
		if duration < 300 { // <5 min
			durationType = "short"
		} else if duration < 600 { // 5-10 min
			durationType = "medium"
		} else {
			durationType = "long"
		}
	}

	// Determine quality from final height value
	quality := ""
	if height > 0 {
		quality = telegram.GetQualityFromResolution(height)
	}

	// Create video document
	now := time.Now()
	videoDoc := models.Video{
		ID:                primitive.NewObjectID(),
		Title:             title,
		Description:       fmt.Sprintf("Batch uploaded: %s", video.Name),
		Thumbnail:         video.ThumbnailURL, // Cloudinary thumbnail URL
		StorageProvider:   models.StorageProviderTelegram,
		TelegramChannelID: channelID,
		TelegramMessageID: result.MessageID,
		TelegramFileID:    result.FileID,
		MimeType:          result.MimeType,
		Duration:          duration,
		DurationType:      durationType,
		Quality:           quality,
		Width:             width,
		Height:            height,
		FileSize:          video.Size,
		Status:            "ready",
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// Insert to MongoDB
	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := services.VideosCollection.InsertOne(dbCtx, videoDoc)
	if err != nil {
		return "", fmt.Errorf("failed to insert video to database: %w", err)
	}

	return videoDoc.ID.Hex(), nil
}

// ============ MAIN UPLOAD PROCESS ============

// Run executes the batch upload process
func (b *BatchUploader) Run(ctx context.Context) error {
	b.startTime = time.Now()

	b.logInfo("========================================")
	b.logInfo("  BATCH VIDEO UPLOADER")
	b.logInfo("========================================")
	b.logInfo("")

	// Scan folder
	if err := b.ScanFolder(); err != nil {
		return err
	}

	if b.totalVideos == 0 {
		b.logWarning("No videos found in %s", VideoFolder)
		return nil
	}

	b.logInfo("")
	b.logInfo("========================================")
	b.logInfo("  STARTING UPLOADS (%d videos)", b.totalVideos)
	b.logInfo("========================================")
	b.logInfo("")

	// Run all uploads inside RunWithCallback to keep MTProto client alive
	b.logInfo("Connecting to Telegram...")
	err := b.telegramService.RunWithCallback(ctx, func(ctx context.Context) error {
		b.logSuccess("Connected to Telegram")
		b.logInfo("")

		// Process videos sequentially - ALL uploads happen inside this callback
		for i, video := range b.videos {
			// Check context
			select {
			case <-ctx.Done():
				b.logWarning("Upload cancelled by user")
				return ctx.Err()
			default:
			}

			b.logInfo("----------------------------------------")
			b.logInfo("Processing video %d/%d", i+1, b.totalVideos)
			b.logInfo("----------------------------------------")

			// Upload video
			if err := b.uploadSingleVideo(ctx, video); err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return err
				}
				b.logError("Failed to upload %s: %v", video.Name, err)
			}

			// Apply delay before next video (except for last video)
			if i < len(b.videos)-1 && video.Uploaded {
				delay := getDelayForSize(video.Size)
				b.logDelay(delay, fmt.Sprintf("after uploading %s file", video.SizeStr))

				select {
				case <-ctx.Done():
					b.logWarning("Upload cancelled during delay")
					return ctx.Err()
				case <-time.After(delay):
					b.logInfo("Delay complete, proceeding to next video")
				}
			}

			b.logInfo("")
		}

		return nil
	})

	// Disconnect after RunWithCallback returns
	b.telegramService.Disconnect()

	if err != nil && !errors.Is(err, context.Canceled) {
		b.printSummary()
		return fmt.Errorf("upload process failed: %w", err)
	}

	b.printSummary()
	return nil
}

// printSummary prints the final upload summary
func (b *BatchUploader) printSummary() {
	totalDuration := time.Since(b.startTime)

	b.logInfo("")
	b.logInfo("========================================")
	b.logInfo("  UPLOAD SUMMARY")
	b.logInfo("========================================")
	b.logInfo("")
	b.logInfo("Total videos found:     %d", b.totalVideos)
	b.logInfo("Successfully uploaded:  %d", b.uploadedCount)
	b.logInfo("Failed:                 %d", b.failedCount)
	b.logInfo("Total data uploaded:    %s", formatSize(b.totalBytesUp))
	b.logInfo("Total time:             %s", totalDuration.Round(time.Second))
	b.logInfo("")
	b.logInfo("FloodWait events:       %d", b.floodWaitEvents)
	b.logInfo("Total FloodWait time:   %s", b.totalWaitTime.Round(time.Second))
	b.logInfo("")

	// List failed videos
	if b.failedCount > 0 {
		b.logInfo("Failed videos:")
		for _, v := range b.videos {
			if v.Error != nil {
				b.logError("  - %s: %v", v.Name, v.Error)
			}
		}
	}

	b.logInfo("========================================")
}

// ============ MAIN ENTRY POINT ============

func printUsage() {
	fmt.Println("Usage: batch_video_uploader [OPTIONS] [FOLDER_PATH]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -h, --help     Show this help message")
	fmt.Println("")
	fmt.Println("Arguments:")
	fmt.Println("  FOLDER_PATH    Path to folder containing videos (default: ./videos)")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  batch_video_uploader                    # Use default ./videos folder")
	fmt.Println("  batch_video_uploader ./my_videos        # Use custom folder")
	fmt.Println("  batch_video_uploader D:/Videos/Upload   # Use absolute path")
	fmt.Println("")
	fmt.Println("Supported formats: mp4, mkv, mov, avi")
	fmt.Println("")
	fmt.Println("Delay rules based on file size:")
	fmt.Println("  < 200 MB       : 1 minute delay")
	fmt.Println("  200 MB - 800 MB: 2 minutes delay")
	fmt.Println("  800 MB - 1.2 GB: 4 minutes delay")
	fmt.Println("  > 1.2 GB       : 6 minutes delay")
}

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Parse command line arguments
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "-h" || arg == "--help" {
			printUsage()
			os.Exit(0)
		}
		// Use provided folder path
		VideoFolder = arg
	}

	fmt.Println("")
	fmt.Println("  Batch Video Uploader for Telegram")
	fmt.Println("  ==================================")
	fmt.Println("")
	fmt.Printf("  Video folder: %s\n", VideoFolder)
	fmt.Println("")

	// Connect to MongoDB
	fmt.Println("  Connecting to MongoDB...")
	if err := services.ConnectDB(); err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	fmt.Println("  MongoDB connected!")
	fmt.Println("")

	// Create uploader
	uploader, err := NewBatchUploader()
	if err != nil {
		log.Fatalf("Failed to create uploader: %v", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nReceived shutdown signal, cancelling uploads...")
		cancel()
	}()

	// Run uploader
	if err := uploader.Run(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			log.Println("Upload process was cancelled")
			os.Exit(0)
		}
		log.Fatalf("Upload failed: %v", err)
	}

	fmt.Println("\nBatch upload completed!")
}
