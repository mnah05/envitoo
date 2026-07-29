package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

func TestResolvePubAck_Ok(t *testing.T) {
	ok := make(chan *jetstream.PubAck, 1)
	ok <- &jetstream.PubAck{}
	errCh := make(chan error)

	if err := resolvePubAck(context.Background(), ok, errCh); err != nil {
		t.Fatalf("resolvePubAck() = %v, want nil", err)
	}
}

func TestResolvePubAck_Err(t *testing.T) {
	ok := make(chan *jetstream.PubAck)
	errCh := make(chan error, 1)
	want := errors.New("publish failed")
	errCh <- want

	got := resolvePubAck(context.Background(), ok, errCh)
	if !errors.Is(got, want) {
		t.Fatalf("resolvePubAck() = %v, want %v", got, want)
	}
}

func TestResolvePubAck_Timeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ok := make(chan *jetstream.PubAck)
	errCh := make(chan error)

	got := resolvePubAck(ctx, ok, errCh)
	if !errors.Is(got, errAckTimeout) {
		t.Fatalf("resolvePubAck() = %v, want %v", got, errAckTimeout)
	}
}

func TestResolvePubAck_PrefersOkOverDeadline(t *testing.T) {
	// Both Ok and Done are ready. Without the nested select, Go's select can
	// pick Done and incorrectly treat a successful publish as timed out.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ok := make(chan *jetstream.PubAck, 1)
	ok <- &jetstream.PubAck{}
	errCh := make(chan error)

	for range 100 {
		// Drain and refill so each iteration sees a ready Ok again.
		select {
		case <-ok:
		default:
		}
		ok <- &jetstream.PubAck{}

		if err := resolvePubAck(ctx, ok, errCh); err != nil {
			t.Fatalf("resolvePubAck() = %v, want nil (Ok must win over Done)", err)
		}
	}
}

func TestResolvePubAck_PrefersErrOverDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ok := make(chan *jetstream.PubAck)
	errCh := make(chan error, 1)
	want := errors.New("nats err")

	for range 100 {
		select {
		case <-errCh:
		default:
		}
		errCh <- want

		got := resolvePubAck(ctx, ok, errCh)
		if !errors.Is(got, want) {
			t.Fatalf("resolvePubAck() = %v, want %v (Err must win over Done)", got, want)
		}
	}
}

func TestResolvePubAck_WaitsUntilOk(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ok := make(chan *jetstream.PubAck)
	errCh := make(chan error)

	go func() {
		time.Sleep(20 * time.Millisecond)
		ok <- &jetstream.PubAck{}
	}()

	if err := resolvePubAck(ctx, ok, errCh); err != nil {
		t.Fatalf("resolvePubAck() = %v, want nil", err)
	}
}
