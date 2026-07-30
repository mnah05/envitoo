package service

import (
	"crypto/subtle"
	"fmt"
	"testing"

	"github.com/google/uuid"
)

func TestGenerateGuestOTP_Format(t *testing.T) {
	for i := 0; i < 50; i++ {
		otp, err := generateGuestOTP()
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		if len(otp) != 6 {
			t.Fatalf("len=%d otp=%q", len(otp), otp)
		}
		for _, c := range otp {
			if c < '0' || c > '9' {
				t.Fatalf("non-digit in %q", otp)
			}
		}
	}
}

func TestGenerateGuestOTP_ZeroPad(t *testing.T) {
	// Format contract used by generateGuestOTP.
	if got := fmt.Sprintf("%06d", 42); got != "000042" {
		t.Fatalf("got %q", got)
	}
}

func TestHmacGuestOTP_StableAndScoped(t *testing.T) {
	secret := []byte("test-secret")
	eventA := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	eventB := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	email := "guest@example.com"
	otp := "000042"

	a1 := hmacGuestOTP(secret, eventA, email, otp)
	a2 := hmacGuestOTP(secret, eventA, email, otp)
	if a1 != a2 {
		t.Fatal("hmac not stable")
	}
	if subtle.ConstantTimeCompare([]byte(a1), []byte(a2)) != 1 {
		t.Fatal("constant-time compare failed for equal digests")
	}

	otherEvent := hmacGuestOTP(secret, eventB, email, otp)
	if a1 == otherEvent {
		t.Fatal("hmac should change with event id")
	}
	otherOTP := hmacGuestOTP(secret, eventA, email, "000043")
	if a1 == otherOTP {
		t.Fatal("hmac should change with otp")
	}
	otherSecret := hmacGuestOTP([]byte("other"), eventA, email, otp)
	if a1 == otherSecret {
		t.Fatal("hmac should change with secret")
	}
}

func TestNormalizeGuestEmail(t *testing.T) {
	got := normalizeGuestEmail("  Guest@Example.COM ")
	if got != "guest@example.com" {
		t.Fatalf("got %q", got)
	}
}

// The Redis key format encodes the event ID and email so keys are scoped
// per event+email pair. There is no cross-event OTP collision possible.
func TestGuestRedisKeys(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	email := "a@b.co"
	if guestOTPKey(id, email) != "guest:otp:aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee:a@b.co" {
		t.Fatal(guestOTPKey(id, email))
	}
	if guestOTPAttemptsKey(id, email) != "guest:otp:attempts:aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee:a@b.co" {
		t.Fatal(guestOTPAttemptsKey(id, email))
	}
	if guestVerifiedKey(id, email) != "guest:verified:aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee:a@b.co" {
		t.Fatal(guestVerifiedKey(id, email))
	}
}
