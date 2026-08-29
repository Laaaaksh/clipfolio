// Package testutil provides a real Postgres-backed *db.Store for
// integration tests, so query logic (SQL, constraints, upserts) is checked
// against actual Postgres rather than mocked away.
package testutil

import (
	"context"
	"os"
	"testing"

	"github.com/laaaaksh/clipfolio/internal/db"
)

// OpenTestStore returns a migrated, freshly-truncated Store backed by the
// database at CLIPFOLIO_TEST_DATABASE_URL. Tests using it are skipped when
// that variable is unset, so `go test ./...` still passes without Postgres
// installed - CI sets it via a service container.
func OpenTestStore(t *testing.T) *db.Store {
	t.Helper()

	url := os.Getenv("CLIPFOLIO_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("CLIPFOLIO_TEST_DATABASE_URL not set, skipping Postgres-backed test")
	}

	ctx := context.Background()
	store, err := db.Open(ctx, url)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(store.Close)

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	if err := truncateAll(ctx, store); err != nil {
		t.Fatalf("truncate test database: %v", err)
	}

	return store
}

func truncateAll(ctx context.Context, store *db.Store) error {
	return store.Exec(ctx, `TRUNCATE TABLE
		cta_clicks, leads, viewer_sessions, lead_gates, ctas, videos, sessions, users
		RESTART IDENTITY CASCADE`)
}
