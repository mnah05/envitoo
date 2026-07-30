package helper

import (
	"regexp"
	"testing"
)

// Note 1: This test uses a map as a set (map[string]struct{}) to detect
// duplicate slugs. struct{} is used as the value type because it occupies
// zero bytes of memory, whereas bool would take 1 byte per entry.

func TestGenerateEventSlug(t *testing.T) {
	// Generate 100 slugs and verify each one meets the expected format.
	// Also check for duplicates — while collisions are astronomically
	// unlikely with a 36^7 space, a bug in the mapping logic could
	// produce a biased output that repeats values.
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
