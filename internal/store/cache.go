package store

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// CachedStore bọc quanh PostgresStore, thêm một lớp cache Redis phía trước
// để tối ưu hiệu năng đọc: hầu hết lượt redirect chỉ cần hit Redis,
// không phải chạm tới PostgreSQL.
type CachedStore struct {
	next *PostgresStore
	rdb  *redis.Client
	ttl  time.Duration
}

// NewCachedStore tạo một CachedStore từ store chính và một client Redis.
func NewCachedStore(next *PostgresStore, rdb *redis.Client, ttl time.Duration) *CachedStore {
	return &CachedStore{next: next, rdb: rdb, ttl: ttl}
}

// Save ghi xuống PostgreSQL trước (đảm bảo tính đúng đắn), sau đó làm nóng
// cache Redis. Nếu ghi cache thất bại, chỉ log lại chứ không làm fail request,
// vì Redis chỉ là lớp tối ưu, không phải nguồn dữ liệu chính.
func (c *CachedStore) Save(ctx context.Context, code, longURL string) error {
	if err := c.next.Save(ctx, code, longURL); err != nil {
		return err
	}
	if err := c.rdb.Set(ctx, cacheKey(code), longURL, c.ttl).Err(); err != nil {
		log.Printf("cảnh báo: không thể ghi cache cho mã %s: %v", code, err)
	}
	return nil
}

// Get đọc theo chiến lược "cache-aside": thử Redis trước, nếu miss (cache
// không có) thì đọc PostgreSQL rồi ghi lại vào cache cho lần sau.
func (c *CachedStore) Get(ctx context.Context, code string) (string, error) {
	val, err := c.rdb.Get(ctx, cacheKey(code)).Result()
	if err == nil {
		return val, nil
	}
	if err != redis.Nil {
		log.Printf("cảnh báo: lỗi đọc Redis cho mã %s: %v", code, err)
	}

	longURL, err := c.next.Get(ctx, code)
	if err != nil {
		return "", err
	}

	if err := c.rdb.Set(ctx, cacheKey(code), longURL, c.ttl).Err(); err != nil {
		log.Printf("cảnh báo: không thể làm nóng cache cho mã %s: %v", code, err)
	}
	return longURL, nil
}

// Exists luôn kiểm tra thẳng ở PostgreSQL để đảm bảo tính duy nhất của mã
// mới, không dựa vào cache (tránh sinh trùng mã do cache chưa đồng bộ).
func (c *CachedStore) Exists(ctx context.Context, code string) (bool, error) {
	return c.next.Exists(ctx, code)
}

// IncrementHits cập nhật số liệu thống kê ở PostgreSQL.
func (c *CachedStore) IncrementHits(ctx context.Context, code string) error {
	return c.next.IncrementHits(ctx, code)
}

func cacheKey(code string) string {
	return "url:" + code
}
