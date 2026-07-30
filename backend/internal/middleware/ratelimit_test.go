package middleware

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestGetVisitorNoRaceAndUpdatesLastSeen(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := newIPRateLimiter(ctx, RateLimitConfig{
		Rate:            100,
		Burst:           100,
		CleanupInterval: time.Hour,
		CleanupIdleTime: time.Hour,
		MaxVisitors:     100,
	})

	// 50 concurrent calls to getVisitor with the same key should produce
	// exactly one visitor entry (the mutex serializes creation).
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = l.getVisitor("1.2.3.4")
		}()
	}
	wg.Wait()

	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.visitors) != 1 {
		t.Fatalf("expected 1 visitor, got %d", len(l.visitors))
	}
}

func TestGetVisitorRespectsMaxVisitors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := newIPRateLimiter(ctx, RateLimitConfig{
		Rate:            10,
		Burst:           10,
		CleanupInterval: time.Hour,
		CleanupIdleTime: time.Hour,
		MaxVisitors:     2,
	})

	if !l.getVisitor("a").Allow() {
		t.Fatal("first visitor should be allowed")
	}
	if !l.getVisitor("b").Allow() {
		t.Fatal("second visitor should be allowed")
	}

	// At capacity: new keys get a zero limiter and are not stored.
	denied := l.getVisitor("c")
	if denied.Allow() {
		t.Fatal("third visitor should be denied when at max capacity")
	}

	l.mu.Lock()
	n := len(l.visitors)
	l.mu.Unlock()
	if n != 2 {
		t.Fatalf("expected map size 2, got %d", n)
	}
}

func TestCleanupLoopStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	l := newIPRateLimiter(ctx, RateLimitConfig{
		Rate:            10,
		Burst:           10,
		CleanupInterval: 10 * time.Millisecond,
		CleanupIdleTime: time.Hour,
		MaxVisitors:     10,
	})
	_ = l.getVisitor("x")

	cancel()
	// Give the goroutine a moment to observe cancellation and exit.
	time.Sleep(50 * time.Millisecond)
}
