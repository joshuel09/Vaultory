//go:build integration

package integration

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/identity"
)

const testSecret = "integration-secret-0123456789abcdef"

func sign(token, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(token))
	return url.QueryEscape(token + "." + base64.StdEncoding.EncodeToString(mac.Sum(nil)))
}

// insertSession creates a session row with explicit lifetimes, so the expiry and cap cases can be
// produced without waiting ninety days.
func insertSession(t *testing.T, userID, token, expires, created string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO "session" ("id","token","userId","expiresAt","createdAt","updatedAt")
		 VALUES ($1,$2,$3, now() + $4::interval, now() - $5::interval, now())`,
		uuid.NewString(), token, userID, expires, created); err != nil {
		t.Fatalf("insert session: %v", err)
	}
}

func resolve(t *testing.T, cookie string) (uuid.UUID, error) {
	t.Helper()
	r, err := http.NewRequest(http.MethodGet, "/api/collectibles", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if cookie != "" {
		r.AddCookie(&http.Cookie{Name: "better-auth.session_token", Value: cookie})
	}
	return identity.NewSessionResolver(pool, testSecret).Resolve(r)
}

// userFor creates an account and returns its id, plus the collector the trigger made.
func userFor(t *testing.T) (string, uuid.UUID) {
	t.Helper()
	id := "verify-" + uuid.NewString()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO "user" ("id","name","email","emailVerified","updatedAt")
		 VALUES ($1,'Verify',$2,false,now())`, id, id+"@example.test"); err != nil {
		t.Fatalf("create account: %v", err)
	}
	var c uuid.UUID
	if err := pool.QueryRow(context.Background(),
		`SELECT id FROM collectors WHERE user_id=$1`, id).Scan(&c); err != nil {
		t.Fatalf("trigger: %v", err)
	}
	return id, c
}

// FR-014, SC-004. Every one of these must be refused, and refused identically: a caller learns
// nothing about which part of their attempt was wrong.
func TestSessionsThatMustBeRefused(t *testing.T) {
	freshStore(t)
	user, _ := userFor(t)

	insertSession(t, user, "live-token", "30 days", "1 hour")
	insertSession(t, user, "expired-token", "-1 minute", "2 days")
	insertSession(t, user, "past-cap-token", "20 days", "91 days")

	// A session whose collector no longer exists. The join returns nothing, which is correct and
	// silent — and silence is exactly why this needs a test. A refactor that turned "no row" into
	// "no filter" would not fail anything else.
	orphanUser, _ := userFor(t)
	insertSession(t, orphanUser, "orphan-token", "30 days", "1 hour")
	if _, err := pool.Exec(context.Background(), `DELETE FROM "user" WHERE "id"=$1`, orphanUser); err != nil {
		t.Fatalf("delete account: %v", err)
	}

	good := sign("live-token", testSecret)
	cases := map[string]string{
		"no cookie at all":            "",
		"expired session":             sign("expired-token", testSecret),
		"past the 90-day cap":         sign("past-cap-token", testSecret),
		"collector no longer exists":  sign("orphan-token", testSecret),
		"unknown token":               sign("no-such-token", testSecret),
		"signature stripped":          "live-token",
		"signature corrupted":         url.QueryEscape("live-token.AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="),
		"signed with another secret":  sign("live-token", "a-different-secret-0123456789abcdef"),
		"token altered after signing": url.QueryEscape("other-token." + good[len(good)-44:]),
	}

	for name, cookie := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := resolve(t, cookie); err != identity.ErrUnresolved {
				t.Fatalf("accepted (%v) — want ErrUnresolved", err)
			}
		})
	}

	// The control. Without this, a resolver that refused everything would pass the whole table.
	t.Run("a valid session resolves", func(t *testing.T) {
		if _, err := resolve(t, good); err != nil {
			t.Fatalf("a valid session was refused: %v", err)
		}
	})
}
