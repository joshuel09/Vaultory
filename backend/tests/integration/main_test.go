//go:build integration

// Package integration exercises Vaultory against a real PostgreSQL instance.
//
// These tests are behind the `integration` build tag because they need a database. Run them with:
//
//	VAULTORY_TEST_DATABASE_URL=postgres://... go test -tags=integration ./tests/integration/...
//
// A real database rather than a fake is the point: the guarantees under test here — the composite
// foreign key, the submission-key uniqueness under concurrency, numeric(12,2) exactness — live in
// the schema, and a fake would assert only that the Go code believes in them.
package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joshuel09/vaultory/backend/internal/collection"
	"github.com/joshuel09/vaultory/backend/internal/imagestore"
	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
)

var pool *pgxpool.Pool

// The two seeded development collectors. Cross-collector isolation cannot be demonstrated with
// one, which is why migration 000005 seeds two.
var (
	collectorA = uuid.MustParse("11111111-1111-4111-8111-111111111111")
	collectorB = uuid.MustParse("22222222-2222-4222-8222-222222222222")
)

func TestMain(m *testing.M) {
	url := os.Getenv("VAULTORY_TEST_DATABASE_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr,
			"VAULTORY_TEST_DATABASE_URL is not set; skipping integration tests.\n"+
				"Start PostgreSQL (docker compose up -d postgres), apply migrations, then set it.")
		os.Exit(0)
	}

	ctx := context.Background()
	var err error
	pool, err = postgres.NewPool(ctx, url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot reach the test database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := ensureSchema(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "test database is not migrated: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// ensureSchema fails loudly rather than running tests against a half-built database, where a
// missing constraint would look like a passing test.
func ensureSchema(ctx context.Context) error {
	for _, table := range []string{"collectors", "collectibles", "collectible_images", "collectible_submissions"} {
		var exists bool
		err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)`,
			table).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("table %q is missing; run the migrations first", table)
		}
	}
	return nil
}

// freshStore clears collector data and returns a store plus service for one test.
//
// Each test starts from an empty collection so ordering and counts are deterministic, and so a
// failure points at the test that caused it rather than at leftovers from another.
func freshStore(t *testing.T) (*postgres.Store, *collection.Service) {
	t.Helper()
	ctx := context.Background()
	for _, stmt := range []string{
		`DELETE FROM collectible_submissions`,
		`UPDATE collectibles SET image_id = NULL`,
		`DELETE FROM collectible_images`,
		`DELETE FROM collectibles`,
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("reset database (%s): %v", stmt, err)
		}
	}
	// Both seeded collectors must exist; every ownership test needs a second party.
	for _, id := range []uuid.UUID{collectorA, collectorB} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO collectors (id, display_name) VALUES ($1, $2) ON CONFLICT (id) DO NOTHING`,
			id, "Test Collector"); err != nil {
			t.Fatalf("seed collector: %v", err)
		}
	}

	store := postgres.NewStore(pool)
	images, err := imagestore.NewFilesystem(t.TempDir())
	if err != nil {
		t.Fatalf("image store: %v", err)
	}
	svc := collection.NewService(store, images, 24*time.Hour)
	// A fixed clock, so the future-date rule is about the rule and not about when the suite ran.
	svc.SetClock(func() time.Time { return time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC) })
	return store, svc
}
