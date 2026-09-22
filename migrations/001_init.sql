-- Bảng lưu ánh xạ mã ngắn -> URL gốc theo mô hình key-value.
CREATE TABLE IF NOT EXISTS urls (
    code       VARCHAR(16) PRIMARY KEY,
    long_url   TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    hits       BIGINT NOT NULL DEFAULT 0
);

-- Index hỗ trợ truy vấn/thống kê theo thời gian tạo.
CREATE INDEX IF NOT EXISTS idx_urls_created_at ON urls (created_at);
