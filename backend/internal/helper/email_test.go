package helper

import "testing"

func TestNormalizeEmail(t *testing.T) {
	// Table-driven test: each entry specifies an input and the expected
	// normalized output. The test covers trimming whitespace, lowercasing,
	// empty input, and already-normal input.
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
