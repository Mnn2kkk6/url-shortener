// Package shortener chịu trách nhiệm sinh mã ngắn (short code) duy nhất.
package shortener

import (
	"crypto/rand"
	"math/big"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Generate tạo một chuỗi mã ngẫu nhiên dạng base62 với độ dài cho trước.
// Dùng crypto/rand để đảm bảo tính ngẫu nhiên an toàn, tránh bị đoán trước mã.
func Generate(length int) (string, error) {
	b := make([]byte, length)
	max := big.NewInt(int64(len(alphabet)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = alphabet[n.Int64()]
	}
	return string(b), nil
}
