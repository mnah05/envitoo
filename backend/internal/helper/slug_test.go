package helper

import (
	"regexp"
	"testing"
)

func TestGenerateEventSlug(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for range 100 {
		slug, err := GenerateEventSlug()
		if err != nil {
			t.Fatalf("GenerateEventSlug() error = %v", err)
		}
		if len(slug) != eventSlugLength {
			t.Fatalf("slug length = %d, want %d (%q)", len(slug), eventSlugLength, slug)
		}
		if !regexp.MustCompile(`^[a-z0-9]{7}$`).MatchString(slug) {
			t.Fatalf("slug %q does not match expected charset", slug)
		}
		if _, ok := seen[slug]; ok {
			t.Fatalf("duplicate slug %q in sample", slug)
		}
		seen[slug] = struct{}{}
	}
}
