//go:build integration

// Package contract asserts that responses conform to contracts/openapi.yaml.
//
// The contract is the source of truth for the seam between the two applications, and the
// frontend's types are generated from it. A response that drifts from it breaks the frontend
// silently, which is what these tests exist to prevent (Constitution Principle III).
package contract

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joshuel09/vaultory/backend/internal/collection"
	"github.com/joshuel09/vaultory/backend/internal/identity"
	"github.com/joshuel09/vaultory/backend/internal/imagestore"
	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
	"github.com/joshuel09/vaultory/backend/internal/transport/httpapi"
)

var pool *pgxpool.Pool

var (
	collectorA = uuid.MustParse("11111111-1111-4111-8111-111111111111")
	collectorB = uuid.MustParse("22222222-2222-4222-8222-222222222222")
)

const devSecret = "contract-test-secret"

func TestMain(m *testing.M) {
	url := os.Getenv("VAULTORY_TEST_DATABASE_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr, "VAULTORY_TEST_DATABASE_URL is not set; skipping contract tests.")
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
	os.Exit(m.Run())
}

type harness struct {
	server *httptest.Server
	dev    *identity.DevResolver
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	ctx := context.Background()
	for _, stmt := range []string{
		`DELETE FROM collectible_submissions`,
		`UPDATE collectibles SET image_id = NULL`,
		`DELETE FROM collectible_images`,
		`DELETE FROM collectibles`,
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("reset: %v", err)
		}
	}
	for _, id := range []uuid.UUID{collectorA, collectorB} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO collectors (id, display_name) VALUES ($1, 'Contract Collector')
			 ON CONFLICT (id) DO NOTHING`, id); err != nil {
			t.Fatalf("seed collector: %v", err)
		}
	}

	images, err := imagestore.NewFilesystem(t.TempDir())
	if err != nil {
		t.Fatalf("image store: %v", err)
	}
	svc := collection.NewService(postgres.NewStore(pool), images, time.Hour)
	svc.SetClock(func() time.Time { return time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC) })

	dev := identity.NewDevResolver(devSecret)
	srv := httptest.NewServer(httpapi.NewServer(svc, dev, dev).Routes())
	t.Cleanup(srv.Close)
	return &harness{server: srv, dev: dev}
}

// sessionFor returns a request cookie for the given collector, obtained the way a browser would:
// from the dev sign-in endpoint. Nothing here forges an identity by hand.
func (h *harness) sessionFor(t *testing.T, collector string) *http.Cookie {
	t.Helper()
	url := h.server.URL + "/api/dev/session"
	if collector != "" {
		url += "?collector=" + collector
	}
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		t.Fatalf("dev session: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	for _, c := range resp.Cookies() {
		if c.Name == identity.SessionCookieName {
			return c
		}
	}
	t.Fatal("dev sign-in returned no session cookie")
	return nil
}

func (h *harness) do(t *testing.T, req *http.Request, cookie *http.Cookie) *http.Response {
	t.Helper()
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := h.server.Client().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", req.Method, req.URL.Path, err)
	}
	return resp
}
