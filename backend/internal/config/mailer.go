package config

import (
	"fmt"
	"strings"

	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
	"github.com/nats-io/nats.go"
)

// MailerConfig holds configuration for the NATS → Resend email consumer.
//
// Note 1: The mailer is a separate process (cmd/mailer) that subscribes to
// NATS for outbound email jobs and sends them through Resend's API. Splitting
// the mailer from the API server means email sending can scale independently
// and a mailer crash won't affect API availability.
type MailerConfig struct {
	NATSURL      string
	ResendAPIKey string
	EmailFrom    string
	AppPublicURL string
}

// LoadMailer reads environment variables for cmd/mailer.
//
// Note 2: This uses an explicit allow-list (switch/case) instead of prefix
// matching. Since there are only four variables, the explicit approach is
// clearer and less error-prone than prefix-based filtering.
//
// Note 3: AppPublicURL is needed so emails render absolute links back to the
// application (e.g. "click here to confirm"). The URL varies between
// environments — localhost in dev, the production domain in prod.
func LoadMailer() (MailerConfig, error) {
	k := koanf.New(".")

	err := k.Load(env.Provider("", ".", func(key string) string {
		switch key {
		case "NATS_URL", "RESEND_API_KEY", "EMAIL_FROM", "APP_PUBLIC_URL":
			return key
		default:
			return ""
		}
	}), nil)
	if err != nil {
		return MailerConfig{}, fmt.Errorf("load mailer env config: %w", err)
	}

	natsURL := k.String("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	// TrimSpace on each string catches accidental whitespace in .env files
	// (e.g. "RESEND_API_KEY = sk_..." with spaces around the = can leave
	// trailing whitespace on the value).
	apiKey := strings.TrimSpace(k.String("RESEND_API_KEY"))
	from := strings.TrimSpace(k.String("EMAIL_FROM"))
	publicURL := strings.TrimSpace(k.String("APP_PUBLIC_URL"))

	if apiKey == "" {
		return MailerConfig{}, fmt.Errorf("RESEND_API_KEY is required")
	}
	if from == "" {
		return MailerConfig{}, fmt.Errorf("EMAIL_FROM is required")
	}
	if publicURL == "" {
		return MailerConfig{}, fmt.Errorf("APP_PUBLIC_URL is required")
	}

	return MailerConfig{
		NATSURL:      natsURL,
		ResendAPIKey: apiKey,
		EmailFrom:    from,
		AppPublicURL: publicURL,
	}, nil
}
