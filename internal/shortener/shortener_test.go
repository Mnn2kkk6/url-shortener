package shortener

import "testing"

func TestGenerateLength(t *testing.T) {
	code, err := Generate(6)
	if err != nil {
		t.Fatalf("không mong đợi lỗi: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("độ dài mã phải là 6, nhận được %d", len(code))
	}
}

func TestGenerateUniqueness(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 2000; i++ {
		code, err := Generate(8)
		if err != nil {
			t.Fatalf("không mong đợi lỗi: %v", err)
		}
		if seen[code] {
			t.Fatalf("phát hiện trùng mã: %s", code)
		}
		seen[code] = true
	}
}
