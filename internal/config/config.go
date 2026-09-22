package config

import (
	"os"
	"strconv"
)

// Config chứa toàn bộ cấu hình của ứng dụng, đọc từ biến môi trường.
type Config struct {
	Port            string
	DatabaseURL     string
	RedisAddr       string
	RedisPassword   string
	BaseURL         string
	CodeLength      int
	CacheTTLSeconds int
}

// Load đọc cấu hình từ biến môi trường, dùng giá trị mặc định nếu thiếu.
func Load() *Config {
	return &Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/urlshortener?sslmode=disable"),
		RedisAddr:       getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:   getEnv("REDIS_PASSWORD", ""),
		BaseURL:         getEnv("BASE_URL", "http://localhost:8080"),
		CodeLength:      getEnvInt("CODE_LENGTH", 6),
		CacheTTLSeconds: getEnvInt("CACHE_TTL_SECONDS", 3600),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
