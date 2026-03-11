package telegram

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// VideoMetadata - Metadata được trích xuất từ video
type VideoMetadata struct {
	Duration      int     `json:"duration"`       // Thời lượng video (giây)
	Width         int     `json:"width"`          // Chiều rộng video
	Height        int     `json:"height"`         // Chiều cao video
	Bitrate       int     `json:"bitrate"`        // Bitrate (kb/s)
	Codec         string  `json:"codec"`          // Video codec
	FrameRate     float64 `json:"frame_rate"`     // FPS
	ThumbnailPath string  `json:"thumbnail_path"` // Đường dẫn thumbnail đã tạo
}

// FFProbeOutput - Cấu trúc output từ ffprobe
type FFProbeOutput struct {
	Format  FFProbeFormat   `json:"format"`
	Streams []FFProbeStream `json:"streams"`
}

// FFProbeFormat - Thông tin format từ ffprobe
type FFProbeFormat struct {
	Duration string `json:"duration"`
	BitRate  string `json:"bit_rate"`
}

// FFProbeStream - Thông tin stream từ ffprobe
type FFProbeStream struct {
	CodecType  string `json:"codec_type"`
	CodecName  string `json:"codec_name"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	RFrameRate string `json:"r_frame_rate"`
}

// ExtractVideoMetadata - Trích xuất metadata từ video file sử dụng ffprobe
func ExtractVideoMetadata(videoPath string) (*VideoMetadata, error) {
	// Kiểm tra ffprobe có sẵn không
	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		// Không có ffprobe - trả về lỗi để caller biết
		return nil, fmt.Errorf("ffprobe not found in PATH: %w", err)
	}

	// Chạy ffprobe để lấy metadata
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		videoPath,
	}

	cmd := exec.Command(ffprobePath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	// Parse JSON output
	var probeOutput FFProbeOutput
	if err := json.Unmarshal(output, &probeOutput); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	metadata := &VideoMetadata{}

	// Lấy duration từ format
	if probeOutput.Format.Duration != "" {
		if duration, err := strconv.ParseFloat(probeOutput.Format.Duration, 64); err == nil {
			metadata.Duration = int(duration)
		}
	}

	// Lấy bitrate từ format
	if probeOutput.Format.BitRate != "" {
		if bitrate, err := strconv.ParseInt(probeOutput.Format.BitRate, 10, 64); err == nil {
			metadata.Bitrate = int(bitrate / 1000) // Convert to kb/s
		}
	}

	// Tìm video stream để lấy thông tin video
	for _, stream := range probeOutput.Streams {
		if stream.CodecType == "video" {
			metadata.Width = stream.Width
			metadata.Height = stream.Height
			metadata.Codec = stream.CodecName

			// Parse frame rate (e.g., "30/1" or "29.97")
			if stream.RFrameRate != "" {
				if strings.Contains(stream.RFrameRate, "/") {
					parts := strings.Split(stream.RFrameRate, "/")
					if len(parts) == 2 {
						num, _ := strconv.ParseFloat(parts[0], 64)
						den, _ := strconv.ParseFloat(parts[1], 64)
						if den > 0 {
							metadata.FrameRate = num / den
						}
					}
				} else {
					metadata.FrameRate, _ = strconv.ParseFloat(stream.RFrameRate, 64)
				}
			}
			break
		}
	}

	return metadata, nil
}

// ExtractThumbnail - Tạo thumbnail từ video sử dụng ffmpeg
// Trả về đường dẫn tới file thumbnail
func ExtractThumbnail(videoPath string, outputDir string) (string, error) {
	// Kiểm tra ffmpeg có sẵn không
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", fmt.Errorf("ffmpeg not found: %w", err)
	}

	// Tạo output directory nếu chưa có
	if outputDir == "" {
		outputDir = os.TempDir()
	}

	// Tạo tên file thumbnail
	baseName := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
	thumbnailPath := filepath.Join(outputDir, baseName+"_thumb.jpg")

	// Chạy ffmpeg để extract thumbnail từ giây thứ 1
	// -ss 1: seek đến giây thứ 1
	// -vframes 1: lấy 1 frame
	// -q:v 2: chất lượng JPEG cao
	args := []string{
		"-y",       // Overwrite output
		"-ss", "1", // Seek to 1 second
		"-i", videoPath,
		"-vframes", "1",
		"-q:v", "2",
		"-vf", "scale=640:-1", // Scale to 640px width, maintain aspect ratio
		thumbnailPath,
	}

	cmd := exec.Command(ffmpegPath, args...)
	if err := cmd.Run(); err != nil {
		// Try without seeking (for very short videos)
		args = []string{
			"-y",
			"-i", videoPath,
			"-vframes", "1",
			"-q:v", "2",
			"-vf", "scale=640:-1",
			thumbnailPath,
		}
		cmd = exec.Command(ffmpegPath, args...)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("ffmpeg thumbnail extraction failed: %w", err)
		}
	}

	// Verify thumbnail was created
	if _, err := os.Stat(thumbnailPath); os.IsNotExist(err) {
		return "", fmt.Errorf("thumbnail file not created")
	}

	return thumbnailPath, nil
}

// ExtractMetadataAndThumbnail - Trích xuất cả metadata và thumbnail
func ExtractMetadataAndThumbnail(videoPath string, thumbnailDir string) (*VideoMetadata, error) {
	// Trích xuất metadata trước
	metadata, err := ExtractVideoMetadata(videoPath)
	if err != nil {
		metadata = &VideoMetadata{} // Fallback to empty if ffprobe fails
	}

	// Tạo thumbnail
	thumbnailPath, err := ExtractThumbnail(videoPath, thumbnailDir)
	if err != nil {
		// Log warning nhưng không fail
		fmt.Printf("Warning: Failed to extract thumbnail: %v\n", err)
	} else {
		metadata.ThumbnailPath = thumbnailPath
	}

	return metadata, nil
}

// GetQualityFromResolution - Xác định chất lượng từ resolution
func GetQualityFromResolution(height int) string {
	switch {
	case height >= 2160:
		return "4K"
	case height >= 1440:
		return "1440p"
	case height >= 1080:
		return "1080p"
	case height >= 720:
		return "720p"
	case height >= 480:
		return "480p"
	case height >= 360:
		return "360p"
	default:
		return "240p"
	}
}

// GetDurationTypeFromSeconds - Xác định loại duration
func GetDurationTypeFromSeconds(duration int) string {
	switch {
	case duration < 300: // < 5 phút
		return "short"
	case duration < 600: // 5-10 phút
		return "medium"
	default: // > 10 phút
		return "long"
	}
}

// IsFFmpegAvailable - Kiểm tra FFmpeg có sẵn không
func IsFFmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

// IsFFprobeAvailable - Kiểm tra FFprobe có sẵn không
func IsFFprobeAvailable() bool {
	_, err := exec.LookPath("ffprobe")
	return err == nil
}
