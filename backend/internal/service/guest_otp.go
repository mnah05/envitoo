package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mnah05/envitoo/internal/cache"
	"github.com/mnah05/envitoo/internal/db"
	"github.com/mnah05/envitoo/internal/dto"
	"github.com/mnah05/envitoo/internal/helper"
)

const (
	guestOTPTTL       = 10 * time.Minute
	guestVerifiedTTL  = 15 * time.Minute
	guestOTPCooldown  = 60 * time.Second
	guestOTPMaxTries  = 5
	guestOTPCodeBound = 1_000_000
)

type guestOTPState struct {
	Hash  string `json:"hash"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type guestVerifiedState struct {
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	VerifiedAt time.Time `json:"verified_at"`
}

type otpEmailTemplateData struct {
	OTP        string `json:"otp"`
	Name       string `json:"name"`
	EventTitle string `json:"event_title"`
	EventID    string `json:"event_id"`
}

// RequestGuestOTP validates the event, generates a 6-digit OTP, stores its
// HMAC in Redis, and queues an email via the outbox table. No registration
// is created at this stage — only after the caller successfully verifies
// the OTP via VerifyGuestOTP and then calls CreateGuestRegistration.
//
// The flow is: request OTP → receive email → verify OTP → create registration.
// A 60-second cooldown prevents resend spam.
func (s *RegistrationService) RequestGuestOTP(ctx context.Context, eventID uuid.UUID, req dto.RequestGuestOTPRequest) (resp dto.RequestGuestOTPResponse, err error) {
	log := s.logger(ctx)
	log.Info("guest_otp_request_start", "event_id", eventID)
	defer func() {
		if err != nil {
			logOpOutcome(log, "guest_otp_request", err, "event_id", eventID)
			return
		}
		log.Info("guest_otp_request_ok", "event_id", eventID)
	}()

	if err := s.requireOTPDeps(); err != nil {
		return dto.RequestGuestOTPResponse{}, err
	}

	email := normalizeGuestEmail(req.Email)
	name := strings.TrimSpace(req.Name)

	event, err := s.queries.GetEventByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dto.RequestGuestOTPResponse{}, fmt.Errorf("event %s: %w", eventID, ErrEventNotFound)
		}
		return dto.RequestGuestOTPResponse{}, fmt.Errorf("get event: %w", err)
	}
	if event.Status != "published" {
		return dto.RequestGuestOTPResponse{}, fmt.Errorf("event %s: %w", eventID, ErrEventNotOpen)
	}

	otpKey := guestOTPKey(eventID, email)
	attemptsKey := guestOTPAttemptsKey(eventID, email)
	cooldownKey := guestOTPCooldownKey(eventID, email)

	ok, err := s.cache.SetNX(ctx, cooldownKey, []byte("1"), guestOTPCooldown)
	if err != nil {
		return dto.RequestGuestOTPResponse{}, fmt.Errorf("otp cooldown: %w", mapCacheErr(err))
	}
	if !ok {
		return dto.RequestGuestOTPResponse{}, ErrOTPRateLimited
	}

	otp, err := generateGuestOTP()
	if err != nil {
		_ = s.cache.DeleteKeys(ctx, cooldownKey)
		return dto.RequestGuestOTPResponse{}, fmt.Errorf("generate otp: %w", err)
	}

	state := guestOTPState{
		Hash:  hmacGuestOTP(s.otpSecret, eventID, email, otp),
		Name:  name,
		Email: email,
	}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		_ = s.cache.DeleteKeys(ctx, cooldownKey)
		return dto.RequestGuestOTPResponse{}, fmt.Errorf("marshal otp state: %w", err)
	}

	// Overwrite any prior OTP for this event+email; reset attempts.
	if err := s.cache.SetBytes(ctx, otpKey, stateBytes, guestOTPTTL); err != nil {
		_ = s.cache.DeleteKeys(ctx, cooldownKey)
		return dto.RequestGuestOTPResponse{}, fmt.Errorf("store otp: %w", mapCacheErr(err))
	}
	_ = s.cache.DeleteKeys(ctx, attemptsKey)

	templateData, err := json.Marshal(otpEmailTemplateData{
		OTP:        otp,
		Name:       name,
		EventTitle: event.Title,
		EventID:    eventID.String(),
	})
	if err != nil {
		_ = s.cache.DeleteKeys(ctx, otpKey, attemptsKey, cooldownKey)
		return dto.RequestGuestOTPResponse{}, fmt.Errorf("marshal template data: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		_ = s.cache.DeleteKeys(ctx, otpKey, attemptsKey, cooldownKey)
		return dto.RequestGuestOTPResponse{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)
	if _, err := qtx.CreateEmailOutbox(ctx, db.CreateEmailOutboxParams{
		Recipient:    email,
		Template:     EmailTemplateOTP,
		TemplateData: templateData,
		Topic:        EmailTopicOTP,
	}); err != nil {
		_ = s.cache.DeleteKeys(ctx, otpKey, attemptsKey, cooldownKey)
		return dto.RequestGuestOTPResponse{}, fmt.Errorf("create email outbox: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		_ = s.cache.DeleteKeys(ctx, otpKey, attemptsKey, cooldownKey)
		return dto.RequestGuestOTPResponse{}, fmt.Errorf("commit outbox: %w", err)
	}

	return dto.RequestGuestOTPResponse{
		Status:           "otp_sent",
		ExpiresInSeconds: int(guestOTPTTL.Seconds()),
	}, nil
}

// VerifyGuestOTP checks the submitted OTP against the HMAC stored in Redis.
// On success it writes a short-lived verified marker and returns a
// verification token. The subsequent CreateGuestRegistration endpoint
// reads this marker to confirm the email was verified.
//
// Brute-force protection: after guestOTPMaxTries (5) failed attempts both
// the OTP state and the attempt counter are deleted, forcing a new OTP
// request.
func (s *RegistrationService) VerifyGuestOTP(ctx context.Context, eventID uuid.UUID, req dto.VerifyGuestOTPRequest) (resp dto.VerifyGuestOTPResponse, err error) {
	log := s.logger(ctx)
	log.Info("guest_otp_verify_start", "event_id", eventID)
	defer func() {
		if err != nil {
			logOpOutcome(log, "guest_otp_verify", err, "event_id", eventID)
			return
		}
		log.Info("guest_otp_verify_ok", "event_id", eventID)
	}()

	if err := s.requireOTPDeps(); err != nil {
		return dto.VerifyGuestOTPResponse{}, err
	}

	email := normalizeGuestEmail(req.Email)
	otp := strings.TrimSpace(req.OTP)

	otpKey := guestOTPKey(eventID, email)
	attemptsKey := guestOTPAttemptsKey(eventID, email)

	raw, err := s.cache.GetBytes(ctx, otpKey)
	if err != nil {
		if errors.Is(err, cache.ErrNotFound) {
			return dto.VerifyGuestOTPResponse{}, ErrOTPInvalid
		}
		return dto.VerifyGuestOTPResponse{}, fmt.Errorf("get otp: %w", mapCacheErr(err))
	}

	count, err := s.cache.Incr(ctx, attemptsKey)
	if err != nil {
		return dto.VerifyGuestOTPResponse{}, fmt.Errorf("incr attempts: %w", mapCacheErr(err))
	}
	if count == 1 {
		// INCR does not set TTL; without Expire the key is permanent.
		if err := s.cache.Expire(ctx, attemptsKey, guestOTPTTL); err != nil && !errors.Is(err, cache.ErrNotFound) {
			s.logger(ctx).WarnContext(ctx, "otp attempts expire failed", "error", err)
		}
	}
	if count > guestOTPMaxTries {
		_ = s.cache.DeleteKeys(ctx, otpKey, attemptsKey)
		return dto.VerifyGuestOTPResponse{}, ErrOTPInvalid
	}

	var state guestOTPState
	if err := json.Unmarshal(raw, &state); err != nil {
		_ = s.cache.DeleteKeys(ctx, otpKey, attemptsKey)
		return dto.VerifyGuestOTPResponse{}, fmt.Errorf("decode otp state: %w", err)
	}

	want := hmacGuestOTP(s.otpSecret, eventID, email, otp)
	if subtle.ConstantTimeCompare([]byte(want), []byte(state.Hash)) != 1 {
		return dto.VerifyGuestOTPResponse{}, ErrOTPInvalid
	}

	_ = s.cache.DeleteKeys(ctx, otpKey, attemptsKey)

	verified, err := json.Marshal(guestVerifiedState{
		Email:      email,
		Name:       state.Name,
		VerifiedAt: time.Now().UTC(),
	})
	if err != nil {
		return dto.VerifyGuestOTPResponse{}, fmt.Errorf("marshal verified state: %w", err)
	}
	if err := s.cache.SetBytes(ctx, guestVerifiedKey(eventID, email), verified, guestVerifiedTTL); err != nil {
		return dto.VerifyGuestOTPResponse{}, fmt.Errorf("store verified: %w", mapCacheErr(err))
	}

	return dto.VerifyGuestOTPResponse{
		Status:           "verified",
		ExpiresInSeconds: int(guestVerifiedTTL.Seconds()),
	}, nil
}

func (s *RegistrationService) requireOTPDeps() error {
	if len(s.otpSecret) == 0 || !s.cache.Available() {
		return ErrDependencyUnavailable
	}
	return nil
}

func mapCacheErr(err error) error {
	if errors.Is(err, cache.ErrUnavailable) {
		return ErrDependencyUnavailable
	}
	return err
}

func normalizeGuestEmail(email string) string {
	return helper.NormalizeEmail(email)
}

func guestOTPKey(eventID uuid.UUID, email string) string {
	return "guest:otp:" + eventID.String() + ":" + email
}

func guestOTPAttemptsKey(eventID uuid.UUID, email string) string {
	return "guest:otp:attempts:" + eventID.String() + ":" + email
}

func guestOTPCooldownKey(eventID uuid.UUID, email string) string {
	return "guest:otp:cooldown:" + eventID.String() + ":" + email
}

func guestVerifiedKey(eventID uuid.UUID, email string) string {
	return "guest:verified:" + eventID.String() + ":" + email
}

// generateGuestOTP returns a cryptographically-random 6-digit string,
// zero-padded on the left. crypto/rand is used (not math/rand) so the
// output is unpredictable — important because the OTP is the only auth
// factor for guest registration.
func generateGuestOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(guestOTPCodeBound))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// hmacGuestOTP computes an HMAC-SHA256 digest of the tuple
// (eventID | email | otp). The secret is the application's OTP_SECRET
// env var. Storing only the digest in Redis (never the plaintext OTP)
// means a Redis compromise does not leak valid OTP values.
func hmacGuestOTP(secret []byte, eventID uuid.UUID, email, otp string) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(eventID.String()))
	_, _ = mac.Write([]byte("|"))
	_, _ = mac.Write([]byte(email))
	_, _ = mac.Write([]byte("|"))
	_, _ = mac.Write([]byte(otp))
	return hex.EncodeToString(mac.Sum(nil))
}
