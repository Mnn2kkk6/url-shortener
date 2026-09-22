// Package store chứa các lớp lưu trữ dữ liệu: PostgreSQL (nguồn chính)
// và Redis (lớp cache đọc) theo mô hình key-value: code -> long_url.
package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound trả về khi không tìm thấy mã rút gọn trong hệ thống.
var ErrNotFound = errors.New("không tìm thấy mã rút gọn")

// PostgresStore là nguồn dữ liệu chính (source of truth) cho URL rút gọn.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore khởi tạo PostgresStore từ một pool kết nối có sẵn.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// Save lưu một cặp (code, longURL) mới vào cơ sở dữ liệu.
func (s *PostgresStore) Save(ctx context.Context, code, longURL string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO urls (code, long_url) VALUES ($1, $2)`,
		code, longURL,
	)
	return err
}

// Exists kiểm tra xem một mã đã tồn tại hay chưa (dùng để tránh trùng mã).
func (s *PostgresStore) Exists(ctx context.Context, code string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM urls WHERE code = $1)`,
		code,
	).Scan(&exists)
	return exists, err
}

// Get đọc URL gốc tương ứng với một mã rút gọn.
func (s *PostgresStore) Get(ctx context.Context, code string) (string, error) {
	var longURL string
	err := s.pool.QueryRow(ctx,
		`SELECT long_url FROM urls WHERE code = $1`,
		code,
	).Scan(&longURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return longURL, nil
}

// IncrementHits tăng bộ đếm lượt truy cập cho một mã, dùng để thống kê.
func (s *PostgresStore) IncrementHits(ctx context.Context, code string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE urls SET hits = hits + 1 WHERE code = $1`,
		code,
	)
	return err
}
