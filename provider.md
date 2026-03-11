# Feature: Migrate Video Storage to Telegram (MTProto)

## 1. Overview

This feature migrates the current video storage system from a mixed provider model (Cloudinary + Mega.nz) to a unified Telegram-based storage architecture using MTProto.

The goal is to simplify infrastructure, improve streaming performance, and eliminate service limitations for large video files.

Images will continue to be stored using Cloudinary.

Videos will be stored in a private Telegram channel.

---

# 2. Current Architecture

## Backend Stack

| Component     | Technology           |
| ------------- | -------------------- |
| Backend       | Go 1.21+             |
| Framework     | Fiber                |
| Database      | MongoDB Atlas        |
| Image Storage | Cloudinary           |
| Video Storage | Cloudinary + Mega.nz |

---

## Current Video Storage Logic

Located in:

```
videoController.go
processVideoUpload()
```

Current decision logic:

```
if durationType == "long" || file.Size > 100MB
    → Mega.nz
else
    → Cloudinary
```

---

## Current Streaming Problem

Mega streaming currently uses:

```
DownloadToWriter()
```

Behavior:

1. Download full file from Mega
2. Store temp file on server
3. Stream temp file to client

Problems:

* High disk I/O
* High RAM usage
* High latency before playback
* Poor scalability

---

# 3. New Storage Architecture

Video storage will be centralized to Telegram using MTProto.

| Media Type | Provider   |
| ---------- | ---------- |
| Images     | Cloudinary |
| Videos     | Telegram   |

Videos will be uploaded to a **private Telegram channel**.

Telegram will act as the storage backend and content delivery source.

---

# 4. Telegram Service Architecture

New service directory:

```
backend/services/telegram/
```

Modules:

```
client.go
uploader.go
streamer.go
types.go
errors.go
```

### client.go

Responsibilities:

* Initialize MTProto client
* Manage session authentication
* Maintain persistent connection

---

### uploader.go

Responsibilities:

* Upload video to Telegram channel
* Handle chunked upload
* Return Telegram metadata

---

### streamer.go

Responsibilities:

* Retrieve video chunks from Telegram
* Support HTTP Range streaming
* Handle concurrent chunk fetching

---

### types.go

Shared structs:

```
TelegramVideoMeta
TelegramUploadProgress
TelegramFileLocation
```

---

### errors.go

Custom errors:

```
ErrTelegramConnection
ErrUploadFailed
ErrInvalidRange
ErrFileNotFound
```

---

# 5. Upload Workflow

Upload pipeline:

1. Client uploads video to backend
2. Backend creates video record (status = processing)
3. Extract metadata using ffprobe
4. Generate thumbnail
5. Upload video to Telegram
6. Store Telegram metadata in database
7. Mark video as ready

---

## Upload Status Tracking

Upload progress is broadcast via WebSocket.

Example struct:

```
type TelegramUploadProgress struct {
    VideoID   string
    Stage     string
    Progress  float64
    BytesSent int64
    TotalSize int64
}
```

Stages:

```
extracting
thumbnail
uploading
processing
completed
```

---

# 6. Streaming Architecture

Streaming endpoint:

```
GET /api/videos/stream/:id
```

Streaming behavior:

1. Client sends HTTP request
2. Browser includes Range header
3. Backend parses Range
4. Backend fetches required bytes from Telegram
5. Backend returns HTTP 206 response

Example:

```
Range: bytes=0-1048576
```

Response:

```
HTTP/1.1 206 Partial Content
Content-Range: bytes 0-1048576/FILE_SIZE
```

---

## Telegram Streaming Method

telegramStreamer will:

1. Map HTTP byte offset
2. Convert to Telegram chunk request
3. Fetch chunk using MTProto
4. Write directly to response stream

Optional optimization:

```
parallel chunk download
```

To improve throughput.

---

# 7. Database Schema Update

Video model extension:

```
type Video struct {

    StorageProvider string

    TelegramChannelID int64
    TelegramMessageID int
    TelegramFileID string

    MimeType string

    VideoURL string
    MegaHash string
}
```

---

## Stream URL Logic

```
func (v *Video) GetStreamURL() string {
    switch v.StorageProvider {
    case "telegram":
        return "/api/videos/stream/" + v.ID
    case "mega":
        return "/api/videos/stream/mega/" + v.MegaHash
    default:
        return v.VideoURL
    }
}
```

---

# 8. Migration Strategy

Existing videos stored on Mega must be migrated.

Migration process:

1. Scan videos with storage_provider = mega
2. Download from Mega
3. Upload to Telegram
4. Update database metadata
5. Mark migration completed

Migration tracking collection:

```
video_migrations
```

States:

```
pending
migrating
completed
failed
```

---

# 9. Controller Refactor

Old controller:

```
CloudinaryService
MegaService
```

New controller:

```
CloudinaryService
TelegramService
```

Example:

```
type VideoController struct {
    cloudinary *services.CloudinaryService
    telegram   *telegram.TelegramService
    uploading  sync.Map
}
```

---

# 10. Benefits

### Infrastructure Simplification

Remove Mega integration.

Single video storage provider.

---

### Streaming Performance

True streaming via Range requests.

Immediate playback.

---

### Storage Limits Removed

Telegram supports files up to 2GB.

Suitable for long videos.

---

### Reduced Server Resource Usage

No temporary file downloads required.

Lower disk and RAM usage.

---

# 11. Risks

Potential limitations:

* Telegram rate limiting
* MTProto connection stability
* Large upload retry handling

Mitigation strategies:

* retry upload mechanism
* connection pooling
* chunk upload resume

---

# 12. Future Improvements

Possible upgrades:

* CDN cache layer
* video transcoding pipeline
* HLS adaptive streaming
* multi-account Telegram upload balancing




thêm layer cache

User
 ↓
CDN (Cloudflare)
 ↓
Stream API
 ↓
Telegram

Giảm 70–90% request Telegram.