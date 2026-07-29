package service

import (
	"errors"
	"log/slog"
	"testing"
)

func TestDomainFailReason(t *testing.T) {
	reason, ok := domainFailReason(ErrEventFull)
	if !ok || reason != "event_full" {
		t.Fatalf("got %q ok=%v", reason, ok)
	}
	if _, ok := domainFailReason(errors.New("boom")); ok {
		t.Fatal("unexpected classification")
	}
}

func TestLogOpOutcome_DoesNotPanic(t *testing.T) {
	log := slog.Default()
	logOpOutcome(log, "registration_create", nil, "event_id", "x")
	logOpOutcome(log, "registration_create", ErrEventFull, "event_id", "x")
	logOpOutcome(log, "registration_create", errors.New("db down"), "event_id", "x")
}
