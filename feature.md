# Feature Design: Migrate Video Storage to Telegram

## Table of Contents

1. [Overview of Current Media Storage System](#1-overview-of-current-media-storage-system)
2. [Problems with the Current Architecture](#2-problems-with-the-current-architecture)
3. [Proposed Telegram-Based Video Storage Architecture](#3-proposed-telegram-based-video-storage-architecture)
4. [Upload Workflow Design](#4-upload-workflow-design)
5. [Streaming Workflow Design](#5-streaming-workflow-design)
6. [Database Schema Updates](#6-database-schema-updates)
7. [New Services or Modules Required](#7-new-services-or-modules-required)
8. [Required Refactors in the Current Codebase](#8-required-refactors-in-the-current-codebase)
9. [Migration Strategy from Existing Video Storage](#9-migration-strategy-from-existing-video-storage)
10. [Risks and Limitations](#10-risks-and-limitations)

---

## 1. Overview of Current Media Storage System

### 1.1 Technology Stack

- **Backend:** Go 1.21+ with Fiber framework
- **Database:** MongoDB Atlas
- **Image Storage:** Cloudinary
- **Video Storage:**
  - Cloudinary for videos < 10 minutes (or < 100MB)
  - Mega.nz for videos >= 10 minutes (or >= 100MB)

### 1.2 Current Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Frontend   │────▶│   Backend    │────▶│  MongoDB     │
│   (React)    │     │   (Go/Fiber) │     │  (Metadata)  │
└──────────────┘     └──────┬───────┘     └──────────────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
              ▼            ▼            ▼
       ┌───────────┐ ┌───────────┐ ┌───────────┐
       │Cloudinary │ │ Cloudinary│ │  Mega.nz  │
       │ (Images)  │ │ (Short    │ │  (Long    │
       │           │ │  Videos)  │ │  Videos)  │
       └───────────┘ └───────────┘ └───────────┘
```

### 1.3 Current Storage Logic

**Location:** `backend/controllers/videoController.go` (lines 330-420)

```go
// Decision logic in processVideoUpload()
if durationType == "long" || file.Size > 100*1024*1024 { // > 100MB → Mega
    // Upload to Mega
    result, err := c.mega.UploadFile(ctx, tempPath, "videos", progressChan)
    storageType = "mega"
} else {
    // Upload to Cloudinary
    result, err := c.cloudinary.UploadVideo(ctx, src, file.Filename, "videos")
    storageType = "cloudinary"
}
```

### 1.4 Current Video Model

**Location:** `backend/models/video.go`

```go
type Video struct {
    ID                 primitive.ObjectID `bson:"_id,omitempty"`
    Title              string             `bson:"title"`
    Description        string             `bson:"description"`
    Thumbnail          string             `bson:"thumbnail"`           // Cloudinary URL
    VideoURL           string             `bson:"video_url"`           // Cloudinary or Mega URL
    CloudinaryPublicID string             `bson:"cloudinary_public_id"`
    MegaHash           string             `bson:"mega_hash"`
    StorageType        string             `bson:"storage_type"`        // cloudinary, mega
    Duration           int                `bson:"duration"`
    DurationType       string             `bson:"duration_type"`
    Quality            string             `bson:"quality"`
    FileSize           int64              `bson:"file_size"`
    // ... other fields
}
```

### 1.5 Current Services

| Service           | File                                    | Purpose                                   |
| ----------------- | --------------------------------------- | ----------------------------------------- |
| CloudinaryService | `backend/services/cloudinaryService.go` | Upload images and short videos            |
| MegaService       | `backend/services/megaService.go`       | Upload long videos with streaming support |

### 1.6 Current Streaming Implementation

**Mega Streaming:** `backend/controllers/videoController.go` (lines 155-190)

```go
// StreamMegaVideo - Stream video từ Mega
// GET /api/videos/stream/mega/:hash
func (c *VideoController) StreamMegaVideo(ctx *fiber.Ctx) error {
    hash := ctx.Params("hash")
    size, err := c.mega.GetFileSize(hash)
    // ... stream implementation
}
```

---

## 2. Problems with the Current Architecture

### 2.1 Dual Storage Complexity

- **Split logic:** Video storage depends on duration/size, creating complexity in upload and retrieval paths
- **Different APIs:** Cloudinary and Mega have different APIs, requiring separate handling
- **Inconsistent streaming:** Different streaming mechanisms for each provider

### 2.2 Service Limitations

| Provider   | Limitation                   | Impact                               |
| ---------- | ---------------------------- | ------------------------------------ |
| Cloudinary | 100MB free tier limit        | Restricts video quality              |
| Cloudinary | 10-minute video limit (free) | Forces split storage                 |
| Mega       | Rate limiting                | Streaming interruptions              |
| Mega       | SDK downloads entire file    | High memory usage, no true streaming |

### 2.3 Mega Streaming Issues

**Current Mega streaming is inefficient:**

```go
// From megaService.go - DownloadToWriter()
// Downloads ENTIRE file to temp before streaming
tempFile, err := os.CreateTemp("", "mega-stream-*.tmp")
err = s.client.DownloadFile(node, tempPath, nil) // Full download
```

- No HTTP Range support
- High disk I/O
- Poor UX for large files

### 2.4 Cost Considerations

- Cloudinary bandwidth costs scale with usage
- No unified CDN integration
- Storage duplication across providers

---

## 3. Proposed Telegram-Based Video Storage Architecture

### 3.1 Architecture Overview

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Frontend   │────▶│   Backend    │────▶│  MongoDB     │
│   (React)    │     │   (Go/Fiber) │     │  (Metadata)  │
└──────────────┘     └──────┬───────┘     └──────────────┘
                           │
              ┌────────────┴────────────┐
              │                         │
              ▼                         ▼
       ┌───────────┐           ┌───────────────────┐
       │Cloudinary │           │    Telegram       │
       │ (Images   │           │ Private Channel   │
       │  ONLY)    │           │   (ALL Videos)    │
       └───────────┘           └───────────────────┘
```

### 3.2 Storage Responsibilities

| Media Type | Storage Provider | Change    |
| ---------- | ---------------- | --------- |
| Images     | Cloudinary       | No change |
| Thumbnails | Cloudinary       | No change |
| ALL Videos | Telegram         | **NEW**   |

### 3.3 Telegram Integration Components

```
backend/services/
├── cloudinaryService.go  (existing - images only)
├── megaService.go        (deprecated - will be removed)
└── telegramService/
    ├── client.go         (MTProto client initialization)
    ├── uploader.go       (Video upload logic)
    ├── streamer.go       (Video streaming with Range support)
    └── types.go          (Telegram-specific types)
```

### 3.4 Component Responsibilities

| Component          | Responsibility                               |
| ------------------ | -------------------------------------------- |
| `telegramClient`   | Initialize and manage MTProto session        |
| `telegramUploader` | Handle chunked uploads to Telegram channel   |
| `telegramStreamer` | Stream video content with HTTP Range support |

### 3.5 Key Design Principles

1. **Single Storage Provider:** All videos go to Telegram (no conditional routing)
2. **Modularity:** TelegramService implements a `VideoStorageProvider` interface
3. **Backward Compatibility:** Existing videos remain accessible during migration
4. **Range Request Support:** True HTTP streaming for video playback

---

## 4. Upload Workflow Design

### 4.1 Workflow Diagram

```
User                Frontend              Backend               Telegram
 │                    │                     │                      │
 ├──[Select Video]───▶│                     │                      │
 │                    ├──[POST /upload]────▶│                      │
 │                    │                     ├──[Validate file]     │
 │                    │                     ├──[Create DB record]  │
 │                    │                     │   (status:processing)│
 │                    │◀───[202 Accepted]───┤                      │
 │                    │                     │                      │
 │                    │     [Background]    │                      │
 │                    │                     ├──[Extract metadata]  │
 │                    │                     ├──[Generate thumbnail]│
 │                    │                     │──[Upload thumbnail]─▶│ Cloudinary
 │                    │                     │                      │
 │                    │                     ├──[Upload video]─────▶│
 │                    │                     │   (chunked MTProto)  │
 │                    │                     │◀──[message_id]───────┤
 │                    │                     │                      │
 │                    │                     ├──[Update DB]         │
 │                    │                     │   (status:ready)     │
 │                    │                     │   (telegram_msg_id)  │
 │◀───[WebSocket]─────┼─────────────────────┤                      │
 │    notification    │                     │                      │
```

### 4.2 Upload Steps (Detailed)

#### Step 1: File Reception

```go
// POST /api/videos/upload
func (c *VideoController) UploadVideo(ctx *fiber.Ctx) error {
    file, err := ctx.FormFile("video")
    // Validate: file type, size limits
    // Create initial video record with status "processing"
    // Return 202 Accepted with video_id
    go c.processVideoUpload(videoID, file, userID)
}
```

#### Step 2: Metadata Extraction

```go
// Using ffprobe or similar
type VideoMetadata struct {
    Duration  int    // seconds
    Width     int
    Height    int
    Codec     string
    Bitrate   int64
    MimeType  string
}
```

#### Step 3: Thumbnail Generation

```go
// Generate thumbnail at 1-second mark
// Upload to Cloudinary (existing flow)
thumbnailURL, _ := cloudinary.UploadImage(ctx, thumbnailFile, "thumbnails")
```

#### Step 4: Telegram Upload

```go
// New: Upload to Telegram channel
result, err := telegramService.UploadVideo(context.Background(), VideoUploadRequest{
    FilePath:    tempFilePath,
    Caption:     video.Title,
    ChannelID:   os.Getenv("TELEGRAM_CHANNEL_ID"),
    ProgressCb:  progressCallback,
})
// result contains: MessageID, FileID, FileSize
```

#### Step 5: Database Update

```go
update := bson.M{
    "$set": bson.M{
        "storage_type":        "telegram",
        "telegram_channel_id": channelID,
        "telegram_message_id": messageID,
        "telegram_file_id":    fileID,
        "file_size":           fileSize,
        "mime_type":           mimeType,
        "duration":            duration,
        "thumbnail":           thumbnailURL,
        "status":              "ready",
    },
}
```

### 4.3 Progress Tracking

```go
type TelegramUploadProgress struct {
    VideoID   string  `json:"video_id"`
    Stage     string  `json:"stage"`     // extracting, uploading, processing
    Progress  float64 `json:"progress"`  // 0-100
    BytesSent int64   `json:"bytes_sent"`
    TotalSize int64   `json:"total_size"`
}
```

---

## 5. Streaming Workflow Design

### 5.1 Workflow Diagram

```
User              Frontend              Backend               Telegram
 │                   │                     │                      │
 ├──[Click Play]────▶│                     │                      │
 │                   ├──[GET /video/:id]──▶│                      │
 │                   │                     ├──[Fetch metadata]    │
 │                   │◀───[video_data]─────┤                      │
 │                   │                     │                      │
 │                   ├──[GET /stream/:id]─▶│                      │
 │                   │   Range: bytes=0-   │                      │
 │                   │                     ├──[Get file from TG]─▶│
 │                   │                     │◀──[file stream]──────┤
 │                   │◀──[206 Partial]─────┤                      │
 │◀──[Video plays]───┤   Content-Range     │                      │
 │                   │                     │                      │
 │   [Seek forward]  │                     │                      │
 │                   ├──[GET /stream/:id]─▶│                      │
 │                   │   Range: bytes=X-Y  │                      │
 │                   │                     ├──[Partial download]─▶│
 │                   │◀──[206 Partial]─────┤                      │
```

### 5.2 Streaming Endpoint Design

```go
// GET /api/videos/stream/:id
func (c *VideoController) StreamVideo(ctx *fiber.Ctx) error {
    videoID := ctx.Params("id")

    // 1. Get video metadata from MongoDB
    video, err := c.getVideoByID(videoID)
    if err != nil {
        return ctx.Status(404).JSON(...)
    }

    // 2. Parse Range header
    rangeHeader := ctx.Get("Range")
    start, end := parseRangeHeader(rangeHeader, video.FileSize)

    // 3. Set response headers
    ctx.Set("Content-Type", video.MimeType)
    ctx.Set("Accept-Ranges", "bytes")
    ctx.Set("Content-Length", fmt.Sprintf("%d", end-start+1))
    ctx.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, video.FileSize))
    ctx.Status(206) // Partial Content

    // 4. Stream from Telegram
    return c.telegram.StreamVideo(ctx, StreamRequest{
        ChannelID:  video.TelegramChannelID,
        MessageID:  video.TelegramMessageID,
        FileID:     video.TelegramFileID,
        Start:      start,
        End:        end,
    })
}
```

### 5.3 Range Request Implementation

```go
func parseRangeHeader(header string, fileSize int64) (start, end int64) {
    if header == "" {
        return 0, fileSize - 1
    }

    // Parse "bytes=START-END"
    // Handle: "bytes=0-", "bytes=0-1000", "bytes=-500"
    // Return validated start/end positions
}
```

### 5.4 Telegram Streaming Service

```go
type StreamRequest struct {
    ChannelID string
    MessageID int
    FileID    string
    Start     int64
    End       int64
}

func (s *TelegramService) StreamVideo(ctx *fiber.Ctx, req StreamRequest) error {
    // Use MTProto to download specific byte range
    // Pipe directly to response writer
    // Support concurrent chunk downloads for performance
}
```

### 5.5 Caching Strategy

```go
// Optional: Cache popular video chunks
type VideoCache struct {
    VideoID   string
    ChunkID   int
    Data      []byte
    ExpiresAt time.Time
}

// Redis or in-memory cache for frequently accessed chunks
```

---

## 6. Database Schema Updates

### 6.1 Updated Video Model

```go
type Video struct {
    // Existing fields
    ID                 primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    Title              string             `json:"title" bson:"title"`
    Description        string             `json:"description" bson:"description"`
    Thumbnail          string             `json:"thumbnail" bson:"thumbnail"`
    Duration           int                `json:"duration" bson:"duration"`
    DurationType       string             `json:"duration_type" bson:"duration_type"`
    Quality            string             `json:"quality" bson:"quality"`
    FileSize           int64              `json:"file_size" bson:"file_size"`
    Tags               []string           `json:"tags" bson:"tags"`
    Genres             []string           `json:"genres" bson:"genres"`
    Views              int64              `json:"views" bson:"views"`
    Likes              int64              `json:"likes" bson:"likes"`
    Status             string             `json:"status" bson:"status"`
    UploadedBy         primitive.ObjectID `json:"uploaded_by" bson:"uploaded_by"`
    CreatedAt          time.Time          `json:"created_at" bson:"created_at"`
    UpdatedAt          time.Time          `json:"updated_at" bson:"updated_at"`

    // Deprecated fields (keep for backward compatibility)
    VideoURL           string             `json:"video_url" bson:"video_url"`
    CloudinaryPublicID string             `json:"cloudinary_public_id" bson:"cloudinary_public_id"`
    MegaHash           string             `json:"mega_hash" bson:"mega_hash"`

    // New Telegram fields
    StorageProvider    string             `json:"storage_provider" bson:"storage_provider"`
    TelegramChannelID  string             `json:"telegram_channel_id" bson:"telegram_channel_id"`
    TelegramMessageID  int                `json:"telegram_message_id" bson:"telegram_message_id"`
    TelegramFileID     string             `json:"telegram_file_id" bson:"telegram_file_id"`
    MimeType           string             `json:"mime_type" bson:"mime_type"`
}
```

### 6.2 Storage Provider Enum

```go
const (
    StorageProviderCloudinary = "cloudinary"  // Legacy
    StorageProviderMega       = "mega"         // Legacy
    StorageProviderTelegram   = "telegram"     // New
)
```

### 6.3 MongoDB Index Updates

```go
// Add indexes for Telegram fields
telegramIndexes := []mongo.IndexModel{
    {
        Keys: bson.D{
            {Key: "storage_provider", Value: 1},
        },
    },
    {
        Keys: bson.D{
            {Key: "telegram_channel_id", Value: 1},
            {Key: "telegram_message_id", Value: 1},
        },
    },
}
```

### 6.4 Migration Status Tracking

```go
// New collection for tracking migration progress
type VideoMigration struct {
    VideoID         primitive.ObjectID `bson:"video_id"`
    SourceProvider  string             `bson:"source_provider"`
    TargetProvider  string             `bson:"target_provider"`
    Status          string             `bson:"status"`  // pending, migrating, completed, failed
    ErrorMessage    string             `bson:"error_message,omitempty"`
    StartedAt       time.Time          `bson:"started_at"`
    CompletedAt     time.Time          `bson:"completed_at,omitempty"`
}
```

---

## 7. New Services or Modules Required

### 7.1 File Structure

```
backend/
├── services/
│   ├── cloudinaryService.go    (existing)
│   ├── megaService.go          (deprecated)
│   ├── database.go             (existing)
│   └── telegram/
│       ├── client.go           (NEW)
│       ├── uploader.go         (NEW)
│       ├── streamer.go         (NEW)
│       ├── types.go            (NEW)
│       └── errors.go           (NEW)
```

### 7.2 TelegramClient (client.go)

```go
package telegram

import (
    "github.com/gotd/td/telegram"
    "github.com/gotd/td/session"
)

type TelegramClient struct {
    client    *telegram.Client
    channelID int64
    mu        sync.Mutex
}

// Config contains Telegram API credentials
type Config struct {
    APIID        int    `env:"TELEGRAM_API_ID"`
    APIHash      string `env:"TELEGRAM_API_HASH"`
    BotToken     string `env:"TELEGRAM_BOT_TOKEN"`    // Optional: for bot mode
    PhoneNumber  string `env:"TELEGRAM_PHONE"`        // For user mode
    SessionPath  string `env:"TELEGRAM_SESSION_PATH"`
    ChannelID    int64  `env:"TELEGRAM_CHANNEL_ID"`
}

func NewTelegramClient(cfg Config) (*TelegramClient, error) {
    // Initialize MTProto client
    // Handle authentication
    // Return configured client
}

func (c *TelegramClient) Connect(ctx context.Context) error
func (c *TelegramClient) Disconnect() error
func (c *TelegramClient) IsConnected() bool
```

### 7.3 TelegramUploader (uploader.go)

```go
package telegram

type VideoUploadRequest struct {
    FilePath   string
    FileName   string
    Caption    string
    ChannelID  int64
    ProgressCb func(progress UploadProgress)
}

type VideoUploadResult struct {
    MessageID  int
    FileID     string
    FileSize   int64
    MimeType   string
    Duration   int
    Width      int
    Height     int
}

type UploadProgress struct {
    BytesUploaded int64
    TotalBytes    int64
    Percent       float64
}

func (c *TelegramClient) UploadVideo(ctx context.Context, req VideoUploadRequest) (*VideoUploadResult, error) {
    // 1. Open file
    // 2. Calculate total size
    // 3. Upload in chunks (512KB each for large files)
    // 4. Send as video message to channel
    // 5. Return message info
}

// Chunked upload for large files (>10MB)
func (c *TelegramClient) uploadLargeFile(ctx context.Context, filePath string, progressCb func(UploadProgress)) (*tg.InputFile, error)
```

### 7.4 TelegramStreamer (streamer.go)

```go
package telegram

type StreamRequest struct {
    ChannelID int64
    MessageID int
    FileID    string
    Start     int64  // Byte offset start
    End       int64  // Byte offset end
}

type StreamResponse struct {
    ContentType   string
    ContentLength int64
    TotalSize     int64
    Reader        io.ReadCloser
}

func (c *TelegramClient) StreamVideo(ctx context.Context, req StreamRequest) (*StreamResponse, error) {
    // 1. Get file location from message
    // 2. Request specific byte range from Telegram
    // 3. Return streaming reader
}

func (c *TelegramClient) GetFileSize(ctx context.Context, channelID int64, messageID int) (int64, error) {
    // Get file size without downloading
}
```

### 7.5 Types and Errors (types.go, errors.go)

```go
package telegram

// types.go
type MessageInfo struct {
    ID        int
    ChannelID int64
    FileID    string
    FileRef   []byte
    FileSize  int64
    MimeType  string
    Duration  int
}

// errors.go
var (
    ErrNotConnected     = errors.New("telegram client not connected")
    ErrMessageNotFound  = errors.New("message not found")
    ErrFileNotFound     = errors.New("file not found in message")
    ErrRangeNotSatisfy  = errors.New("requested range not satisfiable")
    ErrUploadFailed     = errors.New("upload failed")
    ErrAuthRequired     = errors.New("authentication required")
)
```

### 7.6 Storage Provider Interface

```go
// For future extensibility
type VideoStorageProvider interface {
    Upload(ctx context.Context, filePath string, opts UploadOptions) (*UploadResult, error)
    Stream(ctx context.Context, fileID string, start, end int64) (io.ReadCloser, error)
    Delete(ctx context.Context, fileID string) error
    GetFileInfo(ctx context.Context, fileID string) (*FileInfo, error)
}

// TelegramService implements VideoStorageProvider
type TelegramService struct {
    client *TelegramClient
}
```

### 7.7 Environment Variables

```env
# Telegram Configuration (add to .env)
TELEGRAM_API_ID=12345678
TELEGRAM_API_HASH=0123456789abcdef0123456789abcdef
TELEGRAM_PHONE=+84123456789
TELEGRAM_SESSION_PATH=./telegram_session
TELEGRAM_CHANNEL_ID=-1001234567890

# Optional: Bot mode (alternative to user auth)
TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrsTUVwxyz
```

---

## 8. Required Refactors in the Current Codebase

### 8.1 VideoController Refactors

**File:** `backend/controllers/videoController.go`

| Section            | Current                                 | Change Required                  |
| ------------------ | --------------------------------------- | -------------------------------- |
| Constructor        | Creates CloudinaryService + MegaService | Add TelegramService              |
| processVideoUpload | Split logic (Cloudinary vs Mega)        | Single path to Telegram          |
| StreamMegaVideo    | Mega-specific streaming                 | Replace with generic StreamVideo |
| DeleteVideo        | Handles both providers                  | Update for Telegram              |

**Specific Changes:**

```go
// Before
type VideoController struct {
    cloudinary *services.CloudinaryService
    mega       *services.MegaService
    uploading  sync.Map
}

// After
type VideoController struct {
    cloudinary *services.CloudinaryService
    telegram   *telegram.TelegramService
    uploading  sync.Map
}
```

```go
// Before: processVideoUpload()
if durationType == "long" || file.Size > 100*1024*1024 {
    // Upload to Mega
} else {
    // Upload to Cloudinary
}

// After: processVideoUpload()
// All videos go to Telegram
result, err := c.telegram.UploadVideo(ctx, telegram.VideoUploadRequest{...})
```

### 8.2 Routes Refactors

**File:** `backend/routes/routes.go`

```go
// Before
videos.Get("/stream/mega/:hash", videoController.StreamMegaVideo)

// After
videos.Get("/stream/:id", videoController.StreamVideo)  // Generic streaming
```

### 8.3 Model Updates

**File:** `backend/models/video.go`

- Add new Telegram-specific fields
- Rename `StorageType` to `StorageProvider` for clarity
- Add helper methods for backwards compatibility

```go
// Helper method for streaming URL
func (v *Video) GetStreamURL() string {
    switch v.StorageProvider {
    case "telegram":
        return fmt.Sprintf("/api/videos/stream/%s", v.ID.Hex())
    case "cloudinary":
        return v.VideoURL  // Direct Cloudinary URL
    case "mega":
        return fmt.Sprintf("/api/videos/stream/mega/%s", v.MegaHash)
    default:
        return v.VideoURL
    }
}
```

### 8.4 Frontend API Updates

**File:** `frontend/src/services/api.js`

```javascript
// Before
// No streaming endpoint

// After
export const videosAPI = {
  // ... existing methods
  getStreamURL: (id) => `${baseURL}/videos/stream/${id}`,
};
```

### 8.5 Video Player Updates

**File:** `frontend/src/components/Player.jsx`

```jsx
// Update video source to use streaming endpoint
<video src={`/api/videos/stream/${videoId}`} type={video.mime_type} controls />
```

### 8.6 Files to Modify Summary

| File                                     | Type of Change              |
| ---------------------------------------- | --------------------------- |
| `backend/controllers/videoController.go` | Major refactor              |
| `backend/models/video.go`                | Add fields, add methods     |
| `backend/routes/routes.go`               | Add/update routes           |
| `backend/services/database.go`           | Add indexes                 |
| `backend/main.go`                        | Initialize Telegram service |
| `frontend/src/services/api.js`           | Add streaming helpers       |
| `frontend/src/components/Player.jsx`     | Update video source         |
| `frontend/src/pages/VideoDetail.jsx`     | Use new streaming URL       |

---

## 9. Migration Strategy from Existing Video Storage

### 9.1 Migration Phases

```
Phase 1: Preparation (1 week)
├── Implement TelegramService
├── Add new database fields
├── Create migration tracking collection
└── Test upload/stream with new videos

Phase 2: Dual-Write (2 weeks)
├── New uploads go to Telegram only
├── Old videos remain accessible
├── Monitor for issues
└── Build migration tooling

Phase 3: Background Migration (2-4 weeks)
├── Migrate existing videos to Telegram
├── Track progress in migration collection
├── Validate migrated content
└── Update database records

Phase 4: Cleanup (1 week)
├── Remove MegaService code
├── Update Cloudinary to images-only
├── Delete migrated files from old storage
└── Remove deprecated fields
```

### 9.2 Migration Tool

```go
// cmd/migrate/main.go
package main

type MigrationOptions struct {
    BatchSize      int
    Workers        int
    DryRun         bool
    SourceProvider string  // "cloudinary" or "mega"
}

func migrateVideos(opts MigrationOptions) error {
    // 1. Find all videos with source provider
    // 2. For each video in parallel:
    //    a. Download from source
    //    b. Upload to Telegram
    //    c. Update database
    //    d. Mark migration complete
}
```

### 9.3 Migration Workflow

```
┌─────────────────┐
│ Get pending     │
│ videos (batch)  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ For each video: │
│ - Download src  │◄──────────────┐
│ - Upload to TG  │               │
│ - Update DB     │               │
│ - Mark complete │               │
└────────┬────────┘               │
         │                        │
         ▼                        │
    ┌────────┐    No    ┌─────────┴───────┐
    │ Done?  ├─────────▶│ Get next batch  │
    └────┬───┘          └─────────────────┘
         │ Yes
         ▼
┌─────────────────┐
│ Cleanup source  │
│ (optional)      │
└─────────────────┘
```

### 9.4 Rollback Strategy

```go
// If migration fails, system should:
// 1. Keep original video accessible
// 2. Log failure with error details
// 3. Allow manual retry
// 4. Support rollback command to revert DB changes

func rollbackVideo(videoID string) error {
    // Restore original storage_provider
    // Remove telegram_* fields
    // Delete from Telegram channel
}
```

### 9.5 Backward Compatibility During Migration

```go
// StreamVideo should handle all providers
func (c *VideoController) StreamVideo(ctx *fiber.Ctx) error {
    video := getVideo(id)

    switch video.StorageProvider {
    case "telegram":
        return c.streamFromTelegram(ctx, video)
    case "cloudinary":
        return ctx.Redirect(video.VideoURL)
    case "mega":
        return c.streamFromMega(ctx, video)
    default:
        // Legacy: check if mega_hash exists
        if video.MegaHash != "" {
            return c.streamFromMega(ctx, video)
        }
        return ctx.Redirect(video.VideoURL)
    }
}
```

---

## 10. Risks and Limitations

### 10.1 Telegram API Limitations

| Limitation          | Impact                        | Mitigation                          |
| ------------------- | ----------------------------- | ----------------------------------- |
| 2GB file size limit | Cannot store very large files | Compress/transcode before upload    |
| Rate limiting       | Upload speed throttled        | Queue uploads, respect limits       |
| API changes         | Breaking changes possible     | Abstract behind interface           |
| Account ban risk    | TOS violations                | Use dedicated channel, follow rules |

### 10.2 Technical Risks

| Risk               | Probability | Impact | Mitigation                            |
| ------------------ | ----------- | ------ | ------------------------------------- |
| MTProto complexity | High        | Medium | Use gotd/td library, thorough testing |
| Session management | Medium      | High   | Persistent sessions, auto-reconnect   |
| Stream buffering   | Medium      | Medium | Implement chunked downloading         |
| Auth token expiry  | Low         | High   | Auto-refresh mechanism                |

### 10.3 Operational Risks

| Risk                    | Description               | Mitigation                                 |
| ----------------------- | ------------------------- | ------------------------------------------ |
| Single point of failure | Telegram down = no videos | Cache popular videos, graceful degradation |
| Data loss               | Channel deleted           | Regular backups, multi-region              |
| Performance             | Higher latency than CDN   | Edge caching, video transcoding            |

### 10.4 Legal/Compliance Considerations

- Telegram TOS allows file storage but prohibits illegal content
- No SLA guarantees from Telegram
- Data residency may vary by Telegram datacenter
- Consider backup storage provider for critical content

### 10.5 Performance Considerations

| Aspect             | Current    | With Telegram      | Notes                      |
| ------------------ | ---------- | ------------------ | -------------------------- |
| Upload speed       | ~10 MB/s   | ~5-10 MB/s         | Telegram rate limits       |
| First byte latency | 100-300ms  | 200-500ms          | MTProto overhead           |
| Streaming quality  | Good (CDN) | Good               | Range requests work        |
| Concurrent users   | No limit   | Per-channel limits | May need multiple channels |

### 10.6 Recommendations

1. **Start with new uploads only** - Don't migrate existing videos immediately
2. **Implement monitoring** - Track upload success rates, streaming latency
3. **Plan for failover** - Keep ability to switch to alternative storage
4. **Test thoroughly** - Load testing, range requests, concurrent streams
5. **Document operational procedures** - Auth refresh, error handling, scaling

---

## Appendix A: Telegram MTProto Libraries (Go)

| Library            | Stars | Status     | Notes                       |
| ------------------ | ----- | ---------- | --------------------------- |
| gotd/td            | 1.2k+ | Active     | Full MTProto implementation |
| celestix/gotgproto | 200+  | Active     | High-level wrapper          |
| xelaj/mtproto      | 100+  | Maintained | Alternative implementation  |

**Recommended:** `gotd/td` - Most complete and actively maintained

## Appendix B: Required Go Dependencies

```go
// go.mod additions
require (
    github.com/gotd/td v0.91.0
    github.com/gotd/contrib v0.19.0  // Session storage helpers
)
```

## Appendix C: Telegram Channel Setup

1. Create private Telegram channel
2. Get channel ID (starts with -100)
3. Either:
   - **User mode:** Use your account with API_ID/API_HASH
   - **Bot mode:** Create bot, add to channel as admin
4. Store credentials in `.env`

---

_Document Version: 1.0_  
_Last Updated: 2026-03-09_  
_Author: System Architect_
