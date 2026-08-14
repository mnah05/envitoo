package database

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// testDSN returns a database URL suitable for tests. It prefers TEST_DB_URL,
// then falls back to DB_URL only if it points at a local Postgres.
func testDSN(t *testing.T) string {
	t.Helper()
	if dsn := os.Getenv("TEST_DB_URL"); dsn != "" {
		return dsn
	}
	if dsn := os.Getenv("DB_URL"); dsn != "" && isLocalPostgresURL(dsn) {
		return dsn
	}
	t.Skip("set TEST_DB_URL (or local DB_URL) to run database tests")
	return ""
}

func isLocalPostgresURL(u string) bool {
	for _, host := range []string{"localhost", "127.0.0.1", "[::1]", "::1"} {
		if strings.Contains(u, "@"+host+":") ||
			strings.Contains(u, "://"+host+":") ||
			strings.Contains(u, "@"+host+"/") ||
			strings.Contains(u, "://"+host+"/") {
			return true
		}
	}
	return false
}

func TestOpen_Success(t *testing.T) {
	dsn := testDSN(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := Open(ctx, dsn, DefaultPoolConfig())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	if db.Pool() == nil {
		t.Fatal("expected non-nil pool")
	}

	if err := db.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestOpen_InvalidURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := Open(ctx, "not-a-valid-url", DefaultPoolConfig())
	if err == nil {
		t.Fatal("expected Open to fail for invalid URL")
	}
}

func TestOpen_UnreachableHost(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Use a valid-looking URL to a closed port on localhost. The Ping should
	// fail because nothing is listening.
	_, err := Open(ctx, "postgres://user:pass@localhost:54321/db", DefaultPoolConfig())
	if err == nil {
		t.Fatal("expected Open to fail when database is unreachable")
	}
}

func TestOpen_RequireTLS(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := DefaultPoolConfig()
	cfg.RequireTLS = true

	_, err := Open(ctx, "postgres://user:pass@localhost/db?sslmode=disable", cfg)
	if err == nil {
		t.Fatal("expected Open to reject sslmode=disable when TLS is required")
	}
}

func TestClose_ReleasesPool(t *testing.T) {
	dsn := testDSN(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := Open(ctx, dsn, DefaultPoolConfig())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	db.Close()

	// After Close, Ping must fail because the pool is closed.
	if err := db.Ping(ctx); err == nil {
		t.Fatal("expected Ping to fail after Close")
	}
}

func TestStat_AfterOpen(t *testing.T) {
	dsn := testDSN(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := Open(ctx, dsn, DefaultPoolConfig())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	stat := db.Stat()
	if stat == nil {
		t.Fatal("expected non-nil Stat after Open")
	}
}
