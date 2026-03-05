package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/t3rm1n4l/go-mega"
)

// MegaService - Service xử lý upload video dài lên Mega.nz
type MegaService struct {
	client *mega.Mega
	mu     sync.Mutex
}

// MegaUploadResult - Kết quả upload lên Mega
type MegaUploadResult struct {
	FileID    string `json:"file_id"`
	FileName  string `json:"file_name"`
	PublicURL string `json:"public_url"`
	Size      int64  `json:"size"`
}

// MegaUploadProgress - Progress callback
type MegaUploadProgress struct {
	Total    int64 `json:"total"`
	Uploaded int64 `json:"uploaded"`
	Percent  int   `json:"percent"`
}

// NewMegaService - Khởi tạo Mega service và đăng nhập
func NewMegaService() (*MegaService, error) {
	email := os.Getenv("MEGA_EMAIL")
	password := os.Getenv("MEGA_PASSWORD")

	if email == "" || password == "" {
		return nil, fmt.Errorf("missing Mega credentials in environment")
	}

	client := mega.New()
	if err := client.Login(email, password); err != nil {
		return nil, fmt.Errorf("failed to login to Mega: %w", err)
	}

	return &MegaService{client: client}, nil
}

// GetOrCreateFolder - Lấy hoặc tạo folder trên Mega
func (s *MegaService) GetOrCreateFolder(folderPath string) (*mega.Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	root := s.client.FS.GetRoot()

	// Tìm folder theo path
	nodes, err := s.client.FS.PathLookup(root, []string{folderPath})
	if err == nil && len(nodes) > 0 {
		return nodes[len(nodes)-1], nil
	}

	// Tạo folder mới nếu không tồn tại
	newFolder, err := s.client.CreateDir(folderPath, root)
	if err != nil {
		return nil, fmt.Errorf("failed to create folder on Mega: %w", err)
	}

	return newFolder, nil
}

// UploadFile - Upload file lên Mega với progress callback
func (s *MegaService) UploadFile(ctx context.Context, filePath string, folderName string, progressChan chan<- MegaUploadProgress) (*MegaUploadResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Mở file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Lấy thông tin file
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Lấy hoặc tạo folder
	folder, err := s.GetOrCreateFolder(folderName)
	if err != nil {
		// Nếu không tạo được folder, upload vào root
		folder = s.client.FS.GetRoot()
	}

	// Upload file lên Mega
	fileName := filepath.Base(filePath)
	node, err := s.client.UploadFile(filePath, folder, fileName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file to Mega: %w", err)
	}

	// Tạo public link
	publicURL, err := s.client.Link(node, true)
	if err != nil {
		// Nếu không tạo được public link, trả về kết quả không có URL
		return &MegaUploadResult{
			FileID:   node.GetHash(),
			FileName: fileName,
			Size:     fileInfo.Size(),
		}, nil
	}

	return &MegaUploadResult{
		FileID:    node.GetHash(),
		FileName:  fileName,
		PublicURL: publicURL,
		Size:      fileInfo.Size(),
	}, nil
}

// UploadFromReader - Upload từ io.Reader (cho multipart file)
func (s *MegaService) UploadFromReader(ctx context.Context, reader io.Reader, fileName string, size int64, folderName string, progressChan chan<- MegaUploadProgress) (*MegaUploadResult, error) {
	// Lưu tạm file ra disk trước khi upload lên Mega
	// (Mega SDK yêu cầu file path)
	tempDir := os.TempDir()
	tempPath := filepath.Join(tempDir, fileName)

	tempFile, err := os.Create(tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempPath)

	// Copy data vào temp file
	_, err = io.Copy(tempFile, reader)
	tempFile.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}

	// Upload temp file lên Mega
	return s.UploadFile(ctx, tempPath, folderName, progressChan)
}

// DeleteFile - Xóa file trên Mega
func (s *MegaService) DeleteFile(fileHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Tìm node từ hash
	nodes, err := s.client.FS.GetChildren(s.client.FS.GetRoot())
	if err != nil {
		return fmt.Errorf("failed to get children: %w", err)
	}
	for _, node := range nodes {
		if node.GetHash() == fileHash {
			return s.client.Delete(node, true)
		}
	}

	return fmt.Errorf("file not found with hash: %s", fileHash)
}

// GetFileInfo - Lấy thông tin file từ Mega
func (s *MegaService) GetFileInfo(fileHash string) (*mega.Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.findNodeByHash(fileHash)
}

// findNodeByHash - Tìm node từ hash (recursive qua tất cả folders)
func (s *MegaService) findNodeByHash(fileHash string) (*mega.Node, error) {
	var findNode func(parent *mega.Node) *mega.Node
	findNode = func(parent *mega.Node) *mega.Node {
		children, err := s.client.FS.GetChildren(parent)
		if err != nil {
			return nil
		}
		for _, node := range children {
			if node.GetHash() == fileHash {
				return node
			}
			if node.GetType() == mega.FOLDER {
				if found := findNode(node); found != nil {
					return found
				}
			}
		}
		return nil
	}

	root := s.client.FS.GetRoot()
	if found := findNode(root); found != nil {
		return found, nil
	}

	return nil, fmt.Errorf("file not found with hash: %s", fileHash)
}

// GetDownloadURL - Lấy URL download file (share link)
func (s *MegaService) GetDownloadURL(fileHash string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	node, err := s.findNodeByHash(fileHash)
	if err != nil {
		return "", err
	}

	return s.client.Link(node, true)
}

// DownloadToWriter - Stream nội dung file vào io.Writer
func (s *MegaService) DownloadToWriter(fileHash string, w io.Writer) error {
	s.mu.Lock()
	node, err := s.findNodeByHash(fileHash)
	s.mu.Unlock()

	if err != nil {
		return err
	}

	// Tạo temp file để download (go-mega yêu cầu file path)
	tempFile, err := os.CreateTemp("", "mega-stream-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempPath)

	// Download file từ Mega
	s.mu.Lock()
	err = s.client.DownloadFile(node, tempPath, nil)
	s.mu.Unlock()

	if err != nil {
		return fmt.Errorf("failed to download from Mega: %w", err)
	}

	// Đọc và stream ra writer
	file, err := os.Open(tempPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(w, file)
	return err
}

// GetFileSize - Lấy size của file
func (s *MegaService) GetFileSize(fileHash string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	node, err := s.findNodeByHash(fileHash)
	if err != nil {
		return 0, err
	}
	return node.GetSize(), nil
}

// ListFiles - Liệt kê files trong folder
func (s *MegaService) ListFiles(folderName string) ([]*mega.Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	folder, err := s.GetOrCreateFolder(folderName)
	if err != nil {
		return nil, err
	}

	nodes, err := s.client.FS.GetChildren(folder)
	if err != nil {
		return nil, fmt.Errorf("failed to get children: %w", err)
	}
	return nodes, nil
}

// MegaFileInfo - Thông tin file từ Mega để sync
type MegaFileInfo struct {
	Hash       string `json:"hash"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	PublicURL  string `json:"public_url"`
	FolderPath string `json:"folder_path"`
	IsVideo    bool   `json:"is_video"`
}

// ListAllVideos - Liệt kê TẤT CẢ video files từ Mega (recursive)
func (s *MegaService) ListAllVideos() ([]MegaFileInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var videos []MegaFileInfo
	videoExtensions := map[string]bool{
		".mp4": true, ".webm": true, ".mov": true, ".avi": true,
		".mkv": true, ".flv": true, ".wmv": true, ".m4v": true,
	}

	// Recursive function to traverse all folders
	var traverse func(node *mega.Node, path string) error
	traverse = func(node *mega.Node, path string) error {
		children, err := s.client.FS.GetChildren(node)
		if err != nil {
			return err
		}

		for _, child := range children {
			childName := child.GetName()
			childPath := path
			if childPath != "" {
				childPath += "/" + childName
			} else {
				childPath = childName
			}

			if child.GetType() == mega.FOLDER {
				// Traverse into subfolder
				traverse(child, childPath)
			} else if child.GetType() == mega.FILE {
				// Check if it's a video file
				ext := strings.ToLower(filepath.Ext(childName))
				if videoExtensions[ext] {
					// Get public URL
					publicURL, _ := s.client.Link(child, true)

					videos = append(videos, MegaFileInfo{
						Hash:       child.GetHash(),
						Name:       childName,
						Size:       child.GetSize(),
						PublicURL:  publicURL,
						FolderPath: path,
						IsVideo:    true,
					})
				}
			}
		}
		return nil
	}

	// Start from root
	root := s.client.FS.GetRoot()
	if err := traverse(root, ""); err != nil {
		return nil, fmt.Errorf("failed to traverse Mega storage: %w", err)
	}

	return videos, nil
}

// ListAllFiles - Liệt kê TẤT CẢ files từ Mega (recursive) - bao gồm cả ảnh và video
func (s *MegaService) ListAllFiles() ([]MegaFileInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var files []MegaFileInfo

	videoExtensions := map[string]bool{
		".mp4": true, ".webm": true, ".mov": true, ".avi": true,
		".mkv": true, ".flv": true, ".wmv": true, ".m4v": true,
	}
	imageExtensions := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".webp": true, ".bmp": true, ".tiff": true,
	}

	// Recursive function to traverse all folders
	var traverse func(node *mega.Node, path string) error
	traverse = func(node *mega.Node, path string) error {
		children, err := s.client.FS.GetChildren(node)
		if err != nil {
			return err
		}

		for _, child := range children {
			childName := child.GetName()
			childPath := path
			if childPath != "" {
				childPath += "/" + childName
			} else {
				childPath = childName
			}

			if child.GetType() == mega.FOLDER {
				// Traverse into subfolder
				traverse(child, childPath)
			} else if child.GetType() == mega.FILE {
				ext := strings.ToLower(filepath.Ext(childName))
				isVideo := videoExtensions[ext]
				isImage := imageExtensions[ext]

				if isVideo || isImage {
					// Get public URL
					publicURL, _ := s.client.Link(child, true)

					files = append(files, MegaFileInfo{
						Hash:       child.GetHash(),
						Name:       childName,
						Size:       child.GetSize(),
						PublicURL:  publicURL,
						FolderPath: path,
						IsVideo:    isVideo,
					})
				}
			}
		}
		return nil
	}

	// Start from root
	root := s.client.FS.GetRoot()
	if err := traverse(root, ""); err != nil {
		return nil, fmt.Errorf("failed to traverse Mega storage: %w", err)
	}

	return files, nil
}
