package telegram

import (
	"container/list"
	"sync"
	"time"
)

// StreamCache - In-memory LRU cache cho video streaming chunks
// Giúp giảm số lần gọi Telegram API và tăng tốc độ streaming
type StreamCache struct {
	mu       sync.RWMutex
	capacity int                      // Số lượng entries tối đa
	maxSize  int64                    // Kích thước memory tối đa (bytes)
	curSize  int64                    // Kích thước hiện tại
	items    map[string]*list.Element // key -> element
	lru      *list.List               // Danh sách LRU (least recent at back)
	ttl      time.Duration            // Time-to-live cho cache entries
}

// CacheEntry - Một entry trong cache
type CacheEntry struct {
	Key       string
	Data      []byte
	Size      int64
	CreatedAt time.Time
	HitCount  int
}

// CacheKey - Tạo key cho cache từ message ID và byte range
func CacheKey(messageID int, start, end int64) string {
	return formatCacheKey(messageID, start, end)
}

func formatCacheKey(messageID int, start, end int64) string {
	// Round to nearest 1MB boundaries for better cache hits
	chunkSize := int64(DefaultChunkSize)
	startChunk := (start / chunkSize) * chunkSize
	return formatKeyDirect(messageID, startChunk)
}

func formatKeyDirect(messageID int, offset int64) string {
	return stringFormat("%d:%d", messageID, offset)
}

// Simple string format without fmt to reduce allocations
func stringFormat(format string, a ...interface{}) string {
	// Basic implementation for our specific use case
	result := ""
	argIndex := 0
	for i := 0; i < len(format); i++ {
		if format[i] == '%' && i+1 < len(format) && format[i+1] == 'd' {
			if argIndex < len(a) {
				switch v := a[argIndex].(type) {
				case int:
					result += intToString(int64(v))
				case int64:
					result += intToString(v)
				}
				argIndex++
			}
			i++ // Skip 'd'
		} else {
			result += string(format[i])
		}
	}
	return result
}

func intToString(n int64) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

// DefaultStreamCache - Cache mặc định với config hợp lý
var DefaultStreamCache = NewStreamCache(StreamCacheConfig{
	MaxEntries: 1000,              // 1000 chunks
	MaxSize:    512 * 1024 * 1024, // 512 MB max memory
	TTL:        30 * time.Minute,  // 30 phút
})

// StreamCacheConfig - Cấu hình cho stream cache
type StreamCacheConfig struct {
	MaxEntries int           // Số lượng entries tối đa
	MaxSize    int64         // Kích thước memory tối đa (bytes)
	TTL        time.Duration // Time-to-live cho entries
}

// NewStreamCache - Tạo cache mới
func NewStreamCache(config StreamCacheConfig) *StreamCache {
	if config.MaxEntries == 0 {
		config.MaxEntries = 500
	}
	if config.MaxSize == 0 {
		config.MaxSize = 256 * 1024 * 1024 // 256 MB default
	}
	if config.TTL == 0 {
		config.TTL = 15 * time.Minute
	}

	sc := &StreamCache{
		capacity: config.MaxEntries,
		maxSize:  config.MaxSize,
		items:    make(map[string]*list.Element),
		lru:      list.New(),
		ttl:      config.TTL,
	}

	// Start background cleanup goroutine
	go sc.cleanupLoop()

	return sc
}

// Get - Lấy data từ cache
func (c *StreamCache) Get(messageID int, start, end int64) ([]byte, bool) {
	key := CacheKey(messageID, start, end)

	c.mu.RLock()
	elem, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	entry := elem.Value.(*CacheEntry)

	// Check TTL
	if time.Since(entry.CreatedAt) > c.ttl {
		c.Delete(key)
		return nil, false
	}

	// Move to front (most recently used)
	c.mu.Lock()
	c.lru.MoveToFront(elem)
	entry.HitCount++
	c.mu.Unlock()

	return entry.Data, true
}

// Put - Lưu data vào cache
func (c *StreamCache) Put(messageID int, start, end int64, data []byte) {
	if len(data) == 0 {
		return
	}

	key := CacheKey(messageID, start, end)
	size := int64(len(data))

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already exists
	if elem, ok := c.items[key]; ok {
		// Update existing entry
		entry := elem.Value.(*CacheEntry)
		c.curSize -= entry.Size
		entry.Data = data
		entry.Size = size
		entry.CreatedAt = time.Now()
		c.curSize += size
		c.lru.MoveToFront(elem)
		return
	}

	// Evict entries if needed
	for c.curSize+size > c.maxSize || c.lru.Len() >= c.capacity {
		c.evict()
	}

	// Add new entry
	entry := &CacheEntry{
		Key:       key,
		Data:      data,
		Size:      size,
		CreatedAt: time.Now(),
		HitCount:  0,
	}

	elem := c.lru.PushFront(entry)
	c.items[key] = elem
	c.curSize += size
}

// Delete - Xóa entry khỏi cache
func (c *StreamCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.removeElement(elem)
	}
}

// Clear - Xóa toàn bộ cache
func (c *StreamCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.lru.Init()
	c.curSize = 0
}

// Stats - Lấy thống kê cache
func (c *StreamCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CacheStats{
		Entries:     c.lru.Len(),
		MaxEntries:  c.capacity,
		CurrentSize: c.curSize,
		MaxSize:     c.maxSize,
	}
}

// CacheStats - Thống kê cache
type CacheStats struct {
	Entries     int   `json:"entries"`
	MaxEntries  int   `json:"max_entries"`
	CurrentSize int64 `json:"current_size"`
	MaxSize     int64 `json:"max_size"`
}

// evict - Xóa entry ít dùng nhất (internal, must hold lock)
func (c *StreamCache) evict() {
	elem := c.lru.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

// removeElement - Xóa element khỏi cache (internal, must hold lock)
func (c *StreamCache) removeElement(elem *list.Element) {
	entry := elem.Value.(*CacheEntry)
	delete(c.items, entry.Key)
	c.lru.Remove(elem)
	c.curSize -= entry.Size
}

// cleanupLoop - Background loop để cleanup expired entries
func (c *StreamCache) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanupExpired()
	}
}

// cleanupExpired - Xóa các entries đã expired
func (c *StreamCache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	var toRemove []*list.Element

	for elem := c.lru.Back(); elem != nil; elem = elem.Prev() {
		entry := elem.Value.(*CacheEntry)
		if now.Sub(entry.CreatedAt) > c.ttl {
			toRemove = append(toRemove, elem)
		}
	}

	for _, elem := range toRemove {
		c.removeElement(elem)
	}
}

// PrewarmCache - Preload cache cho video (tải trước vài chunks đầu)
func (c *StreamCache) PrewarmCache(messageID int, chunks [][]byte) {
	offset := int64(0)
	for _, chunk := range chunks {
		if len(chunk) == 0 {
			continue
		}
		chunkSize := int64(len(chunk))
		c.Put(messageID, offset, offset+chunkSize-1, chunk)
		offset += chunkSize
	}
}
