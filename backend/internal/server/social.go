package server

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/mnah05/envitoo/internal/db"
	"github.com/mnah05/envitoo/internal/middleware"
	"github.com/mnah05/envitoo/internal/service"
)

// PrerenderEventSocial serves a static HTML shell with Open Graph and
// Twitter card meta tags to social crawlers that don't execute JavaScript
// (WhatsApp, iMessage, Telegram, Facebook, X, LinkedIn, Slack...).
// Non-bot visitors are redirected back to the SPA, which restores the
// original /e/:slug path via the ?next= query param (see main.tsx).
func (s *Server) PrerenderEventSocial(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	slug := c.Param("slug")
	if slug == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "slug is required")
	}

	if !isSocialBot(c.Request().UserAgent()) {
		return c.Redirect(http.StatusTemporaryRedirect, "/?next="+url.PathEscape("/e/"+slug))
	}

	ctx := c.Request().Context()
	event, err := s.events.GetViewableBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "event not found")
		}
		log.ErrorContext(ctx, "social prerender event", "slug", slug, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	html, err := renderEventPrerender(c.Request(), event)
	if err != nil {
		log.ErrorContext(ctx, "render social prerender", "slug", slug, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	c.Response().Header().Set(echo.HeaderContentType, "text/html; charset=utf-8")
	_, err = c.Response().Write([]byte(html))
	return err
}

// isSocialBot reports whether the user agent belongs to a crawler that
// fetches pages without rendering JavaScript. curl is included so the
// prerender can be verified from a terminal.
var socialBotPattern = regexp.MustCompile(
	`(?i)(bot|crawler|spider|slurp|facebookexternalhit|whatsapp|telegram|twitterbot|linkedinbot|` +
		`slackbot|discordbot|applebot|bingpreview|duckduckgo|embedly|outbrain|viber|skypeuripreview|` +
		`pinterest|snapchat|flipboard|tumblr|yandex|baidu|seznam|sogou|exabot|ia_archiver|curl|wget|okhttp)`)

func isSocialBot(userAgent string) bool {
	return socialBotPattern.MatchString(userAgent)
}

type eventPrerenderData struct {
	Title       string
	Description string
	Image       string
	URL         string
	StartTime   string
}

const eventPrerenderTemplate = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{ .Title }}</title>
<meta name="description" content="{{ .Description }}">
<link rel="canonical" href="{{ .URL }}">
<meta property="og:site_name" content="EnviToo">
<meta property="og:type" content="event">
<meta property="og:title" content="{{ .Title }}">
<meta property="og:description" content="{{ .Description }}">
<meta property="og:url" content="{{ .URL }}">
<meta property="og:image" content="{{ .Image }}">
<meta property="og:start_time" content="{{ .StartTime }}">
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="{{ .Title }}">
<meta name="twitter:description" content="{{ .Description }}">
<meta name="twitter:image" content="{{ .Image }}">
</head>
<body></body>
</html>
`

var eventPrerenderTmpl = template.Must(template.New("event-prerender").Parse(eventPrerenderTemplate))

func renderEventPrerender(r *http.Request, event db.Event) (string, error) {
	desc := event.Description
	if desc == nil || strings.TrimSpace(*desc) == "" {
		fallback := fmt.Sprintf("Join %s on EnviToo.", event.Title)
		desc = &fallback
	}
	descText := strings.Join(strings.Fields(*desc), " ")
	if len(descText) > 300 {
		descText = descText[:300] + "…"
	}

	scheme := "https"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	eventURL := scheme + "://" + r.Host + "/e/" + event.Slug

	image := "/og-default.png"
	if event.CoverImageUrl != nil && *event.CoverImageUrl != "" {
		image = *event.CoverImageUrl
	} else {
		image = scheme + "://" + r.Host + image
	}

	var sb strings.Builder
	err := eventPrerenderTmpl.Execute(&sb, eventPrerenderData{
		Title:       event.Title + " — EnviToo",
		Description: descText,
		Image:       image,
		URL:         eventURL,
		StartTime:   event.StartsAt.Format(time.RFC3339),
	})
	return sb.String(), err
}
