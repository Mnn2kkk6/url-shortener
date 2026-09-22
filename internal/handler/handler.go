// Package handler chứa các HTTP handler cho API rút gọn URL và redirect.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"

	"urlshortener/internal/shortener"
	"urlshortener/internal/store"
)

// Store là interface mà handler cần, giúp dễ dàng thay thế/giả lập khi test.
type Store interface {
	Save(ctx context.Context, code, longURL string) error
	Get(ctx context.Context, code string) (string, error)
	Exists(ctx context.Context, code string) (bool, error)
	IncrementHits(ctx context.Context, code string) error
}

// Handler chứa các dependency cần thiết để xử lý request.
type Handler struct {
	store   Store
	baseURL string
	codeLen int
}

// New khởi tạo một Handler mới.
func New(s Store, baseURL string, codeLen int) *Handler {
	return &Handler{store: s, baseURL: baseURL, codeLen: codeLen}
}

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
	LongURL  string `json:"long_url"`
}

// maxGenerateAttempts giới hạn số lần thử sinh mã mới khi bị trùng,
// tránh vòng lặp vô hạn trong trường hợp cực hiếm.
const maxGenerateAttempts = 5

// ShortenURL xử lý POST /api/shorten
// Body: {"url": "https://example.com/duong-dan-rat-dai"}
func (h *Handler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	var req shortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body JSON không hợp lệ")
		return
	}

	if !isValidURL(req.URL) {
		writeError(w, http.StatusBadRequest, "url không hợp lệ, phải bắt đầu bằng http:// hoặc https://")
		return
	}

	ctx := r.Context()

	var code string
	for attempt := 0; attempt < maxGenerateAttempts; attempt++ {
		candidate, err := shortener.Generate(h.codeLen)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "không thể tạo mã")
			return
		}
		exists, err := h.store.Exists(ctx, candidate)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "lỗi cơ sở dữ liệu")
			return
		}
		if !exists {
			code = candidate
			break
		}
	}

	if code == "" {
		writeError(w, http.StatusInternalServerError, "không thể tạo mã duy nhất, vui lòng thử lại")
		return
	}

	if err := h.store.Save(ctx, code, req.URL); err != nil {
		log.Printf("lỗi khi lưu url: %v", err)
		writeError(w, http.StatusInternalServerError, "không thể lưu url")
		return
	}

	resp := shortenResponse{
		Code:     code,
		ShortURL: h.baseURL + "/" + code,
		LongURL:  req.URL,
	}
	writeJSON(w, http.StatusCreated, resp)
}

// Redirect xử lý GET /{code}: tìm URL gốc và chuyển hướng (301).
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" || code == "favicon.ico" {
		http.NotFound(w, r)
		return
	}

	longURL, err := h.store.Get(r.Context(), code)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "không tìm thấy mã rút gọn")
			return
		}
		log.Printf("lỗi khi đọc url: %v", err)
		writeError(w, http.StatusInternalServerError, "lỗi nội bộ")
		return
	}

	// Cập nhật lượt truy cập bất đồng bộ để không làm chậm redirect.
	go func(code string) {
		bgCtx := context.Background()
		if err := h.store.IncrementHits(bgCtx, code); err != nil {
			log.Printf("lỗi khi cập nhật hits: %v", err)
		}
	}(code)

	http.Redirect(w, r, longURL, http.StatusMovedPermanently)
}

func isValidURL(raw string) bool {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
