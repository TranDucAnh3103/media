package telegram

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/gotd/td/tg"
)

const (
	// Telegram API yêu cầu:
	// - limit phải là bội của 4KB (4096 bytes) và tối đa 1MB
	// - offset phải là bội của 1MB cho file lớn (>1MB)
	// - Một số nguồn nói offset cần là bội của 256KB hoặc chunk size

	// Sử dụng 256KB chunk để đảm bảo tương thích tốt nhất
	DefaultChunkSize = 256 * 1024  // 256KB - an toàn cho mọi loại file
	MaxChunkSize     = 1024 * 1024 // 1MB - giới hạn API

	// Alignment requirement: offset phải là bội của giá trị này
	OffsetAlignment = 256 * 1024 // 256KB alignment
)

// TelegramStreamer - Xử lý streaming video từ Telegram
type TelegramStreamer struct {
	client         *TelegramClient
	circuitBreaker *CircuitBreaker
	mu             sync.Mutex
}

// NewTelegramStreamer - Tạo instance streamer mới
func NewTelegramStreamer(client *TelegramClient, cb *CircuitBreaker) *TelegramStreamer {
	return &TelegramStreamer{
		client:         client,
		circuitBreaker: cb,
	}
}

// GetFileSize - Lấy kích thước file từ message
func (s *TelegramStreamer) GetFileSize(ctx context.Context, messageID int) (int64, error) {
	if !s.client.IsConnected() {
		return 0, ErrNotConnected
	}

	var size int64

	err := s.client.ExecuteInConnection(ctx, func(ctx context.Context) error {
		api := s.client.GetAPI()
		if api == nil {
			return ErrNotConnected
		}

		// Lấy message
		messages, err := api.ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
			Channel: s.client.GetInputChannel(),
			ID:      []tg.InputMessageClass{&tg.InputMessageID{ID: messageID}},
		})

		if err != nil {
			return fmt.Errorf("failed to get message: %w", err)
		}

		channelMessages, ok := messages.(*tg.MessagesChannelMessages)
		if !ok || len(channelMessages.Messages) == 0 {
			return ErrMessageNotFound
		}

		msg, ok := channelMessages.Messages[0].(*tg.Message)
		if !ok {
			return ErrMessageNotFound
		}

		media, ok := msg.Media.(*tg.MessageMediaDocument)
		if !ok {
			return ErrFileNotFound
		}

		doc, ok := media.Document.(*tg.Document)
		if !ok {
			return ErrFileNotFound
		}

		size = doc.Size
		return nil
	})

	if err != nil {
		return 0, err
	}

	return size, nil
}

// StreamVideo - Stream video từ Telegram với hỗ trợ Range request và caching
func (s *TelegramStreamer) StreamVideo(ctx context.Context, req StreamRequest, writer io.Writer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.client.IsConnected() {
		return ErrNotConnected
	}
	
	if s.circuitBreaker != nil && !s.circuitBreaker.AllowRequest() {
		return fmt.Errorf("circuit breaker is OPEN, stream request blocked")
	}

	return s.client.ExecuteInConnection(ctx, func(ctx context.Context) error {
		api := s.client.GetAPI()
		if api == nil {
			return ErrNotConnected
		}

		// Lấy document từ message
		doc, err := s.getDocument(ctx, api, req.MessageID)
		if err != nil {
			return err
		}

		// Validate range
		if req.Start < 0 {
			req.Start = 0
		}
		if req.End <= 0 || req.End >= doc.Size {
			req.End = doc.Size - 1
		}
		if req.Start > req.End {
			return ErrInvalidRange
		}

		// Tạo input file location
		location := &tg.InputDocumentFileLocation{
			ID:            doc.ID,
			AccessHash:    doc.AccessHash,
			FileReference: doc.FileReference,
		}

		// Căn chỉnh offset xuống bội của DefaultChunkSize
		// Telegram yêu cầu offset phải aligned
		alignedStart := (req.Start / int64(DefaultChunkSize)) * int64(DefaultChunkSize)
		skipBytes := req.Start - alignedStart // Số bytes cần skip ở chunk đầu

		offset := alignedStart
		bytesWritten := int64(0)
		totalToWrite := req.End - req.Start + 1

		for bytesWritten < totalToWrite {
			chunkSize := int64(DefaultChunkSize)

			// Kiểm tra cache trước
			chunkEnd := offset + chunkSize - 1
			var chunkData []byte

			if cached, found := DefaultStreamCache.Get(req.MessageID, offset, chunkEnd); found {
				chunkData = cached
			} else {
				// Retry with exponential backoff
				boConf := BackoffConfig{
					BaseDelay:         500 * time.Millisecond,
					MaxDelay:          10 * time.Second,
					MaxRetries:        3,
					Multiplier:        2.0,
					JitterFactor:      0.2,
					ResetAfterSuccess: true,
				}
				backoff := NewExponentialBackoff(boConf)
				opID := fmt.Sprintf("stream_%d_offset_%d", req.MessageID, offset)

				err := backoff.Execute(ctx, opID, func() error {
					if s.circuitBreaker != nil && !s.circuitBreaker.AllowRequest() {
						return fmt.Errorf("circuit breaker is OPEN")
					}

					// Download từ Telegram
					file, dlErr := api.UploadGetFile(ctx, &tg.UploadGetFileRequest{
						Location:     location,
						Offset:       offset,
						Limit:        int(chunkSize),
						Precise:      true,
						CDNSupported: false,
					})

					if dlErr != nil {
						if s.circuitBreaker != nil {
							s.circuitBreaker.RecordFailure()
						}
						// Let backoff handle retries
						return fmt.Errorf("chunk download failed: %w", dlErr)
					}

					fileResult, ok := file.(*tg.UploadFile)
					if !ok {
						return ErrChunkDownloadFailed
					}

					chunkData = fileResult.Bytes
					if s.circuitBreaker != nil {
						s.circuitBreaker.RecordSuccess()
					}
					return nil
				})

				if err != nil {
					return fmt.Errorf("failed to download chunk at offset %d after retries: %w", offset, err)
				}

				// Cache chunk
				if len(chunkData) > 0 {
					DefaultStreamCache.Put(req.MessageID, offset, chunkEnd, chunkData)
				}
			}

			if len(chunkData) == 0 {
				// End of file
				break
			}

			// Xác định phần dữ liệu cần ghi
			dataToWrite := chunkData

			// Skip bytes ở chunk đầu tiên (nếu start không aligned)
			if skipBytes > 0 {
				if skipBytes >= int64(len(dataToWrite)) {
					skipBytes -= int64(len(dataToWrite))
					offset += chunkSize
					continue
				}
				dataToWrite = dataToWrite[skipBytes:]
				skipBytes = 0
			}

			// Trim nếu vượt quá số bytes cần ghi
			remainingToWrite := totalToWrite - bytesWritten
			if int64(len(dataToWrite)) > remainingToWrite {
				dataToWrite = dataToWrite[:remainingToWrite]
			}

			// Write to response
			n, err := writer.Write(dataToWrite)
			if err != nil {
				return fmt.Errorf("failed to write chunk: %w", err)
			}

			bytesWritten += int64(n)
			offset += chunkSize

			// Check context cancellation
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}

		return nil
	})
}

// getDocument - Lấy document từ message
func (s *TelegramStreamer) getDocument(ctx context.Context, api *tg.Client, messageID int) (*tg.Document, error) {
	// Lấy message
	messages, err := api.ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
		Channel: s.client.GetInputChannel(),
		ID:      []tg.InputMessageClass{&tg.InputMessageID{ID: messageID}},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	channelMessages, ok := messages.(*tg.MessagesChannelMessages)
	if !ok || len(channelMessages.Messages) == 0 {
		return nil, ErrMessageNotFound
	}

	msg, ok := channelMessages.Messages[0].(*tg.Message)
	if !ok {
		return nil, ErrMessageNotFound
	}

	media, ok := msg.Media.(*tg.MessageMediaDocument)
	if !ok {
		return nil, ErrFileNotFound
	}

	doc, ok := media.Document.(*tg.Document)
	if !ok {
		return nil, ErrFileNotFound
	}

	return doc, nil
}

// GetVideoMetadata - Lấy metadata của video
func (s *TelegramStreamer) GetVideoMetadata(ctx context.Context, messageID int) (*TelegramVideoMeta, error) {
	if !s.client.IsConnected() {
		return nil, ErrNotConnected
	}

	var meta *TelegramVideoMeta

	err := s.client.RunWithClient(ctx, func(ctx context.Context) error {
		api := s.client.GetAPI()
		if api == nil {
			return ErrNotConnected
		}

		doc, err := s.getDocument(ctx, api, messageID)
		if err != nil {
			return err
		}

		meta = &TelegramVideoMeta{
			FileID:   fmt.Sprintf("%d", doc.ID),
			FileSize: doc.Size,
			MimeType: doc.MimeType,
		}

		// Lấy video attributes
		for _, attr := range doc.Attributes {
			if videoAttr, ok := attr.(*tg.DocumentAttributeVideo); ok {
				meta.Width = videoAttr.W
				meta.Height = videoAttr.H
				meta.Duration = int(videoAttr.Duration)
				break
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return meta, nil
}

// ParseRangeHeader - Parse Range header từ HTTP request
func ParseRangeHeader(rangeHeader string, fileSize int64) (start, end int64, err error) {
	// Default: trả về toàn bộ file
	if rangeHeader == "" {
		return 0, fileSize - 1, nil
	}

	// Parse "bytes=START-END"
	var startStr, endStr string

	// Tìm phần "bytes="
	if len(rangeHeader) < 6 || rangeHeader[:6] != "bytes=" {
		return 0, fileSize - 1, nil
	}

	rangeSpec := rangeHeader[6:]

	// Tìm dấu "-"
	for i := 0; i < len(rangeSpec); i++ {
		if rangeSpec[i] == '-' {
			startStr = rangeSpec[:i]
			endStr = rangeSpec[i+1:]
			break
		}
	}

	// Parse start
	if startStr == "" {
		// bytes=-500 (lấy 500 bytes cuối)
		if endStr != "" {
			end, err = strconv.ParseInt(endStr, 10, 64)
			if err != nil {
				return 0, 0, ErrInvalidRange
			}
			start = fileSize - end
			end = fileSize - 1
			return start, end, nil
		}
		return 0, fileSize - 1, nil
	}

	start, err = strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		return 0, 0, ErrInvalidRange
	}

	// Parse end
	if endStr == "" {
		// bytes=500- (từ byte 500 đến cuối)
		end = fileSize - 1
	} else {
		end, err = strconv.ParseInt(endStr, 10, 64)
		if err != nil {
			return 0, 0, ErrInvalidRange
		}
	}

	// Validate range
	if start < 0 {
		start = 0
	}
	if end >= fileSize {
		end = fileSize - 1
	}
	if start > end {
		return 0, 0, ErrRangeNotSatisfiable
	}

	return start, end, nil
}
