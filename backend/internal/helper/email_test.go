package helper

import "testing"

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"  Guest@Example.COM ", "guest@example.com"},
		{"", ""},
		{"a@b.co", "a@b.co"},
	}
	for _, tt := range tests {
		if got := NormalizeEmail(tt.in); got != tt.want {
			t.Fatalf("NormalizeEmail(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
