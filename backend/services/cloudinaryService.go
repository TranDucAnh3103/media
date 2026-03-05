package services

import (
	"context"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/admin"
	"github.com/cloudinary/cloudinary-go/v2/api/admin/search"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

// CloudinaryService - Service xử lý upload ảnh/video lên Cloudinary
type CloudinaryService struct {
	client *cloudinary.Cloudinary
	mu     sync.Mutex
}

// CloudinaryUploadResult - Kết quả upload
type CloudinaryUploadResult struct {
	PublicID  string `json:"public_id"`
	URL       string `json:"url"`
	SecureURL string `json:"secure_url"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Duration  int    `json:"duration"` // Cho video (giây)
	Format    string `json:"format"`
	Bytes     int64  `json:"bytes"`
}

// NewCloudinaryService - Khởi tạo Cloudinary service
func NewCloudinaryService() (*CloudinaryService, error) {
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		return nil, fmt.Errorf("missing Cloudinary credentials in environment")
	}

	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloudinary client: %w", err)
	}

	return &CloudinaryService{client: cld}, nil
}

// UploadImage - Upload một ảnh lên Cloudinary
// Tự động resize ảnh lớn, tối ưu chất lượng
func (s *CloudinaryService) UploadImage(ctx context.Context, file multipart.File, filename string, folder string) (*CloudinaryUploadResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Cấu hình upload với transformation
	uploadParams := uploader.UploadParams{
		Folder:         folder,
		PublicID:       strings.TrimSuffix(filename, filepath.Ext(filename)),
		ResourceType:   "image",
		Transformation: "c_limit,w_1920,h_2560,q_auto:good,f_auto", // Resize nếu quá lớn
	}

	result, err := s.client.Upload.Upload(ctx, file, uploadParams)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	return &CloudinaryUploadResult{
		PublicID:  result.PublicID,
		URL:       result.URL,
		SecureURL: result.SecureURL,
		Width:     result.Width,
		Height:    result.Height,
		Format:    result.Format,
		Bytes:     int64(result.Bytes),
	}, nil
}

// UploadImages - Upload nhiều ảnh song song (multi-thread với Goroutines)
func (s *CloudinaryService) UploadImages(ctx context.Context, files []*multipart.FileHeader, folder string) ([]CloudinaryUploadResult, error) {
	results := make([]CloudinaryUploadResult, len(files))
	errors := make([]error, len(files))
	var wg sync.WaitGroup

	// Upload song song với goroutines, giới hạn 5 concurrent
	sem := make(chan struct{}, 5)

	for i, fileHeader := range files {
		wg.Add(1)
		go func(index int, fh *multipart.FileHeader) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			file, err := fh.Open()
			if err != nil {
				errors[index] = err
				return
			}
			defer file.Close()

			result, err := s.UploadImage(ctx, file, fh.Filename, folder)
			if err != nil {
				errors[index] = err
				return
			}
			results[index] = *result
		}(i, fileHeader)
	}

	wg.Wait()

	// Kiểm tra lỗi
	for _, err := range errors {
		if err != nil {
			return results, err
		}
	}

	return results, nil
}

// UploadVideo - Upload video lên Cloudinary (chỉ cho video ngắn <10 phút)
func (s *CloudinaryService) UploadVideo(ctx context.Context, file multipart.File, filename string, folder string) (*CloudinaryUploadResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Cấu hình upload video với chuyển đổi format tối ưu
	eagerAsync := true
	uploadParams := uploader.UploadParams{
		Folder:       folder,
		PublicID:     strings.TrimSuffix(filename, filepath.Ext(filename)) + "_" + fmt.Sprintf("%d", time.Now().Unix()),
		ResourceType: "video",
		// Eager transformations - chuyển đổi video sau khi upload
		EagerAsync: &eagerAsync,
	}

	result, err := s.client.Upload.Upload(ctx, file, uploadParams)
	if err != nil {
		return nil, fmt.Errorf("failed to upload video: %w", err)
	}

	// Duration - lấy từ video info nếu có
	duration := 0

	return &CloudinaryUploadResult{
		PublicID:  result.PublicID,
		URL:       result.URL,
		SecureURL: result.SecureURL,
		Width:     result.Width,
		Height:    result.Height,
		Duration:  duration,
		Format:    result.Format,
		Bytes:     int64(result.Bytes),
	}, nil
}

// UploadVideoFromPath - Upload video từ file path
func (s *CloudinaryService) UploadVideoFromPath(ctx context.Context, filePath string, folder string) (*CloudinaryUploadResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := filepath.Base(filePath)
	eagerAsync := true
	uploadParams := uploader.UploadParams{
		Folder:       folder,
		PublicID:     strings.TrimSuffix(filename, filepath.Ext(filename)) + "_" + fmt.Sprintf("%d", time.Now().Unix()),
		ResourceType: "video",
		EagerAsync:   &eagerAsync,
	}

	result, err := s.client.Upload.Upload(ctx, filePath, uploadParams)
	if err != nil {
		return nil, fmt.Errorf("failed to upload video from path: %w", err)
	}

	// Duration - retrieved separately if needed
	duration := 0

	return &CloudinaryUploadResult{
		PublicID:  result.PublicID,
		URL:       result.URL,
		SecureURL: result.SecureURL,
		Width:     result.Width,
		Height:    result.Height,
		Duration:  duration,
		Format:    result.Format,
		Bytes:     int64(result.Bytes),
	}, nil
}

// DeleteResource - Xóa resource từ Cloudinary
func (s *CloudinaryService) DeleteResource(ctx context.Context, publicID string, resourceType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.client.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID:     publicID,
		ResourceType: resourceType,
	})
	return err
}

// DeleteFolder - Xóa toàn bộ folder và resources trong đó từ Cloudinary
// Ví dụ: folder = "comics/abc123" sẽ xóa tất cả ảnh trong folder đó
func (s *CloudinaryService) DeleteFolder(ctx context.Context, folder string, resourceType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Xóa tất cả resources trong folder với prefix
	_, err := s.client.Admin.DeleteAssetsByPrefix(ctx, admin.DeleteAssetsByPrefixParams{
		Prefix: []string{folder},
	})
	if err != nil {
		return fmt.Errorf("failed to delete resources in folder %s: %w", folder, err)
	}

	// Xóa folder rỗng
	_, err = s.client.Admin.DeleteFolder(ctx, admin.DeleteFolderParams{
		Folder: folder,
	})
	// Ignore error if folder doesn't exist
	return nil
}

// GenerateThumbnail - Tạo thumbnail từ video
func (s *CloudinaryService) GenerateThumbnail(publicID string) string {
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	// Tạo thumbnail từ frame đầu tiên của video
	return fmt.Sprintf("https://res.cloudinary.com/%s/video/upload/so_1,w_640,h_360,c_fill/%s.jpg",
		cloudName, publicID)
}

// GetStreamingURL - Lấy URL streaming cho video (HLS/DASH)
func (s *CloudinaryService) GetStreamingURL(publicID string, format string) string {
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	switch format {
	case "hls":
		return fmt.Sprintf("https://res.cloudinary.com/%s/video/upload/sp_hd/%s.m3u8", cloudName, publicID)
	case "dash":
		return fmt.Sprintf("https://res.cloudinary.com/%s/video/upload/sp_hd/%s.mpd", cloudName, publicID)
	default:
		return fmt.Sprintf("https://res.cloudinary.com/%s/video/upload/%s", cloudName, publicID)
	}
}

// CloudinaryResource - Thông tin resource từ Cloudinary
type CloudinaryResource struct {
	PublicID  string `json:"public_id"`
	URL       string `json:"url"`
	SecureURL string `json:"secure_url"`
	Type      string `json:"type"`
	Format    string `json:"format"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Bytes     int64  `json:"bytes"`
}

// ListResources - Liệt kê tất cả resources trong folder theo resource type
func (s *CloudinaryService) ListResources(ctx context.Context, folder string, resourceType string) ([]CloudinaryResource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	maxResults := 500
	params := admin.AssetsParams{
		Prefix:     folder,
		MaxResults: maxResults,
	}

	result, err := s.client.Admin.Assets(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	// Video formats
	videoFormats := map[string]bool{"mp4": true, "webm": true, "mov": true, "avi": true, "mkv": true, "flv": true, "wmv": true}

	var resources []CloudinaryResource
	for _, asset := range result.Assets {
		// Determine type by format
		isVideo := videoFormats[asset.Format]
		assetType := "image"
		if isVideo {
			assetType = "video"
		}

		// Filter by requested type
		if resourceType != "" && assetType != resourceType {
			continue
		}

		resources = append(resources, CloudinaryResource{
			PublicID:  asset.PublicID,
			URL:       asset.URL,
			SecureURL: asset.SecureURL,
			Type:      assetType,
			Format:    asset.Format,
			Width:     asset.Width,
			Height:    asset.Height,
			Bytes:     int64(asset.Bytes),
		})
	}

	return resources, nil
}

// ListAllResources - Liệt kê TẤT CẢ resources từ Cloudinary (cả image và video)
func (s *CloudinaryService) ListAllResources(ctx context.Context) (images []CloudinaryResource, videos []CloudinaryResource, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Sử dụng Admin API để lấy tất cả images (đáng tin cậy hơn Search API)
	// Search API có thể có delay khi index images mới

	// Lấy tất cả images bằng Admin Assets API
	maxResults := 500
	imageParams := admin.AssetsParams{
		MaxResults: maxResults,
	}
	imageResult, adminErr := s.client.Admin.Assets(ctx, imageParams)
	if adminErr == nil && imageResult != nil {
		for _, asset := range imageResult.Assets {
			images = append(images, CloudinaryResource{
				PublicID:  asset.PublicID,
				URL:       asset.URL,
				SecureURL: asset.SecureURL,
				Type:      "image",
				Format:    asset.Format,
				Width:     asset.Width,
				Height:    asset.Height,
				Bytes:     int64(asset.Bytes),
			})
		}
	}

	// Nếu Admin API không trả về nhiều, thử thêm Search API
	if len(images) < 10 {
		imageSearchResult, searchErr := s.client.Admin.Search(ctx, search.Query{
			Expression: "resource_type:image",
			MaxResults: 500,
		})
		if searchErr == nil && imageSearchResult != nil {
			// Thêm images chưa có
			existingIDs := make(map[string]bool)
			for _, img := range images {
				existingIDs[img.PublicID] = true
			}
			for _, asset := range imageSearchResult.Assets {
				if !existingIDs[asset.PublicID] {
					images = append(images, CloudinaryResource{
						PublicID:  asset.PublicID,
						URL:       asset.URL,
						SecureURL: asset.SecureURL,
						Type:      "image",
						Format:    asset.Format,
						Width:     asset.Width,
						Height:    asset.Height,
						Bytes:     int64(asset.Bytes),
					})
				}
			}
		}
	}

	// Lấy TẤT CẢ VIDEOS từ Cloudinary bằng Search API
	videoResult, searchErr := s.client.Admin.Search(ctx, search.Query{
		Expression: "resource_type:video",
		MaxResults: 500,
	})
	if searchErr == nil && videoResult != nil {
		for _, asset := range videoResult.Assets {
			videos = append(videos, CloudinaryResource{
				PublicID:  asset.PublicID,
				URL:       asset.URL,
				SecureURL: asset.SecureURL,
				Type:      "video",
				Format:    asset.Format,
				Width:     asset.Width,
				Height:    asset.Height,
				Bytes:     int64(asset.Bytes),
			})
		}
	}

	return images, videos, nil
}

// ListImagesInFolder - Liệt kê tất cả images trong một folder cụ thể (recursive)
func (s *CloudinaryService) ListImagesInFolder(ctx context.Context, folderPrefix string) ([]CloudinaryResource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var images []CloudinaryResource

	// Dùng Admin API với prefix (đáng tin cậy hơn Search API)
	maxResults := 500
	params := admin.AssetsParams{
		Prefix:     folderPrefix,
		MaxResults: maxResults,
	}

	result, err := s.client.Admin.Assets(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("admin assets failed: %w", err)
	}

	for _, asset := range result.Assets {
		images = append(images, CloudinaryResource{
			PublicID:  asset.PublicID,
			URL:       asset.URL,
			SecureURL: asset.SecureURL,
			Type:      "image",
			Format:    asset.Format,
			Width:     asset.Width,
			Height:    asset.Height,
			Bytes:     int64(asset.Bytes),
		})
	}

	return images, nil
}

// ListAllFolders - Liệt kê tất cả folders
func (s *CloudinaryService) ListAllFolders(ctx context.Context) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.client.Admin.RootFolders(ctx, admin.RootFoldersParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}

	var folders []string
	for _, folder := range result.Folders {
		folders = append(folders, folder.Name)
	}

	return folders, nil
}

// ListSubFolders - Liệt kê các subfolder trong một folder
func (s *CloudinaryService) ListSubFolders(ctx context.Context, parentFolder string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.client.Admin.SubFolders(ctx, admin.SubFoldersParams{
		Folder: parentFolder,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list subfolders: %w", err)
	}

	var folders []string
	for _, folder := range result.Folders {
		folders = append(folders, folder.Path)
	}

	return folders, nil
}
