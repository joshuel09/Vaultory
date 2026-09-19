//go:build integration

// Package contract asserts that responses conform to contracts/openapi.yaml.
//
// The contract is the source of truth for the seam between the two applications, and the
// frontend's types are generated from it. A response that drifts from it breaks the frontend
// silently, which is what these tests exist to prevent (Constitution Principle III).
package contract

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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

const devSecret = "contract-test-secret-0123456789abcdef"

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
	server     *httptest.Server
	collectorA uuid.UUID
	collectorB uuid.UUID
	userA      string
	userB      string
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
	userA, collectorA := newAccount(t)
	userB, collectorB := newAccount(t)

	images, err := imagestore.NewFilesystem(t.TempDir())
	if err != nil {
		t.Fatalf("image store: %v", err)
	}
	svc := collection.NewService(postgres.NewStore(pool), images, time.Hour)
	svc.SetClock(func() time.Time { return time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC) })

	resolver := identity.NewSessionResolver(pool, devSecret)
	srv := httptest.NewServer(httpapi.NewServer(svc, resolver).Routes())
	t.Cleanup(srv.Close)
	return &harness{
		server: srv, collectorA: collectorA, collectorB: collectorB,
		userA: userA, userB: userB,
	}
}

// newAccount creates an account and returns it with the collector the trigger made for it.
func newAccount(t *testing.T) (string, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	userID := "contract-" + uuid.NewString()
	if _, err := pool.Exec(ctx,
		`INSERT INTO "user" ("id", "name", "email", "emailVerified", "updatedAt")
		 VALUES ($1, 'Contract Collector', $2, false, now())`,
		userID, userID+"@example.test"); err != nil {
		t.Fatalf("create account: %v", err)
	}
	var id uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM collectors WHERE user_id = $1`, userID).Scan(&id); err != nil {
		t.Fatalf("trigger did not create a collector: %v", err)
	}
	return userID, id
}

// signCookie reproduces Better Auth's cookie format: the token, a dot, and the HMAC over it in
// standard base64, URL-encoded. Verified against a library-generated fixture in the unit suite.
func signCookie(token, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(token))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return url.QueryEscape(token + "." + sig)
}

// sessionFor returns a cookie for one of the harness's collectors.
//
// The session row is inserted and the cookie signed here, because the development sign-in endpoint
// it used to call no longer exists — that is the feature. Nothing forges an identity: the cookie
// is exactly what Better Auth would issue, and the server verifies it the same way either way.
func (h *harness) sessionFor(t *testing.T, collector string) *http.Cookie {
	t.Helper()
	user := h.userA
	if collector == "second" {
		user = h.userB
	}
	token := "tok-" + uuid.NewString()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO "session" ("id", "token", "userId", "expiresAt", "updatedAt")
		 VALUES ($1, $2, $3, now() + interval '30 days', now())`,
		uuid.NewString(), token, user); err != nil {
		t.Fatalf("create session: %v", err)
	}
	return &http.Cookie{Name: "better-auth.session_token", Value: signCookie(token, devSecret)}
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
