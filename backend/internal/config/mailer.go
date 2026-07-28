package config

import (
	"fmt"
	"strings"

	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
	"github.com/nats-io/nats.go"
)

// MailerConfig holds configuration for the NATS → Resend email consumer.
type MailerConfig struct {
	NATSURL      string
	ResendAPIKey string
	EmailFrom    string
	AppPublicURL string
}

// LoadMailer reads environment variables for cmd/mailer.
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
