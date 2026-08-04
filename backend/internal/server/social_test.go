package server

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mnah05/envitoo/internal/db"
)

func TestIsSocialBot(t *testing.T) {
	cases := []struct {
		name string
		ua   string
		want bool
	}{
		{"whatsapp", "WhatsApp/2.23.17.76 A", true},
		{"facebook", "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)", true},
		{"telegram", "TelegramBot (like TwitterBot)", true},
		{"slack", "Slackbot-LinkExpanding 1.0 (+https://api.slack.com/robots)", true},
		{"googlebot", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", true},
		{"curl", "curl/8.4.0", true},
		{"chrome", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36", false},
		{"safari-mobile", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 Version/17.5 Mobile/15E148 Safari/604.1", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isSocialBot(tc.ua); got != tc.want {
				t.Fatalf("isSocialBot(%q) = %v, want %v", tc.ua, got, tc.want)
			}
		})
	}
}

func TestRenderEventPrerender(t *testing.T) {
	desc := `Design meetup for the <community>.
Second line with "quotes".`
	event := db.Event{
		ID:            uuid.New(),
		Title:         `Launch & <Party>`,
		Description:   &desc,
		StartsAt:      time.Date(2026, 8, 15, 18, 0, 0, 0, time.UTC),
		Slug:          "launch-party",
		CoverImageUrl: strPtr("https://img.example.com/cover.png"),
	}

	req := httptest.NewRequest("GET", "/e/launch-party", nil)
	req.Host = "envitoo.example.com"
	req.Header.Set("X-Forwarded-Proto", "https")

	html, err := renderEventPrerender(req, event)
	if err != nil {
		t.Fatalf("renderEventPrerender: %v", err)
	}

	for _, want := range []string{
		`<title>Launch &amp; &lt;Party&gt; — EnviToo</title>`,
		`content="Design meetup for the &lt;community&gt;. Second line with &#34;quotes&#34;."`,
		`content="https://img.example.com/cover.png"`,
		`content="https://envitoo.example.com/e/launch-party"`,
		`content="2026-08-15T18:00:00Z"`,
		`og:type" content="event">`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("prerender HTML missing %q\n%s", want, html)
		}
	}
}

func TestRenderEventPrerenderFallbacks(t *testing.T) {
	event := db.Event{
		ID:       uuid.New(),
		Title:    "No cover event",
		StartsAt: time.Now().UTC(),
		Slug:     "no-cover",
	}

	req := httptest.NewRequest("GET", "/e/no-cover", nil)
	req.Host = "envitoo.example.com"

	html, err := renderEventPrerender(req, event)
	if err != nil {
		t.Fatalf("renderEventPrerender: %v", err)
	}

	if !strings.Contains(html, `content="https://envitoo.example.com/og-default.png"`) {
		t.Fatalf("expected fallback og:image, got:\n%s", html)
	}
	if !strings.Contains(html, `Join No cover event on EnviToo.`) {
		t.Fatalf("expected fallback description, got:\n%s", html)
	}
}

func strPtr(s string) *string { return &s }
