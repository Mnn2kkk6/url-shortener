package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"urlshortener/internal/config"
	"urlshortener/internal/handler"
	"urlshortener/internal/store"
)

func main() {
	// Nạp file .env nếu có (bỏ qua lỗi nếu không tìm thấy)
	_ = godotenv.Load()

	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Kết nối PostgreSQL (nguồn dữ liệu chính - source of truth)
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("không thể kết nối PostgreSQL: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping PostgreSQL thất bại: %v", err)
	}
	log.Println("đã kết nối PostgreSQL")

	// Kết nối Redis (lớp cache đọc)
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("ping Redis thất bại: %v", err)
	}
	defer rdb.Close()
	log.Println("đã kết nối Redis")

	pgStore := store.NewPostgresStore(pool)
	cachedStore := store.NewCachedStore(pgStore, rdb, time.Duration(cfg.CacheTTLSeconds)*time.Second)

	h := handler.New(cachedStore, cfg.BaseURL, cfg.CodeLength)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	// Trang chủ: giao diện web đơn giản để nhập link cần rút gọn.
	// Lưu ý: chạy `go run ./cmd/server` từ thư mục gốc dự án để tìm đúng web/index.html.
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "web/index.html")
	})
	mux.HandleFunc("POST /api/shorten", h.ShortenURL)
	mux.HandleFunc("GET /{code}", h.Redirect)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("server đang chạy tại cổng :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("lỗi server: %v", err)
		}
	}()

	// Chờ tín hiệu tắt (Ctrl+C) để graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("đang tắt server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("lỗi khi tắt server: %v", err)
	}
	log.Println("server đã tắt xong")
}