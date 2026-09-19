package identity

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Better Auth's default session cookie, and the name it uses when the site is served over HTTPS.
// Both are checked because the same binary runs in both places.
const (
	sessionCookieName       = "better-auth.session_token"
	secureSessionCookieName = "__Secure-better-auth.session_token"
)

// sessionAbsoluteCap bounds how long any session may live, however actively it is used.
//
// Better Auth extends expires_at as a session is used, which gives the sliding window FR-010 asks
// for, but it has no notion of an absolute ceiling. This is the ceiling, and it is enforced here
// because this is where verification happens — a cap applied in the frontend would be a cap the
// backend does not know about (research Decision 6).
const sessionAbsoluteCap = 90 * 24 * time.Hour

// SessionResolver establishes the acting collector by verifying a Better Auth session.
//
// It reaches its own conclusion from evidence it can check: it recomputes the cookie's signature
// with the shared secret, and it confirms the session exists and is live by querying PostgreSQL.
// It never calls the Next.js application, and it never reads an identity from a header, body, or
// query parameter (FR-013).
type SessionResolver struct {
	pool   *pgxpool.Pool
	secret []byte
}

func NewSessionResolver(pool *pgxpool.Pool, secret string) *SessionResolver {
	return &SessionResolver{pool: pool, secret: []byte(secret)}
}

// Resolve verifies the session cookie and returns the collector it belongs to.
//
// Every failure returns ErrUnresolved, deliberately. A caller cannot distinguish an absent cookie
// from a forged signature from an expired session, so an attacker probing the endpoint learns
// nothing about which part of their guess was wrong.
func (s *SessionResolver) Resolve(r *http.Request) (CollectorID, error) {
	raw := cookieValue(r)
	if raw == "" {
		return uuid.Nil, ErrUnresolved
	}

	token, ok := s.verify(raw)
	if !ok {
		return uuid.Nil, ErrUnresolved
	}

	collector, err := s.lookup(r.Context(), token)
	if err != nil {
		return uuid.Nil, err
	}
	return collector, nil
}

func cookieValue(r *http.Request) string {
	for _, name := range []string{sessionCookieName, secureSessionCookieName} {
		if c, err := r.Cookie(name); err == nil && c.Value != "" {
			return c.Value
		}
	}
	return ""
}

// verify checks the cookie's signature and returns the token inside it.
//
// The format is Better Auth's, via better-call: urlencode(token + "." + signature), where the
// signature is HMAC-SHA256 over the token, encoded with *standard* base64 — `btoa`, not base64url.
// Feature 001's development cookie used RawURLEncoding for its own signature, and copying that
// pattern here would fail every verification (contracts/README.md).
func (s *SessionResolver) verify(raw string) (string, bool) {
	decoded, err := url.QueryUnescape(raw)
	if err != nil {
		// Not an error worth distinguishing: a cookie that will not decode cannot be verified.
		decoded = raw
	}

	// The last separator, not the first. Base64 padding can contain "=" but never ".", and
	// splitting on the first would break the moment a token format contains one.
	i := strings.LastIndex(decoded, ".")
	if i <= 0 || i == len(decoded)-1 {
		return "", false
	}
	token, signature := decoded[:i], decoded[i+1:]

	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(token))
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// Constant time: a signature check that leaks timing is a signature check that can be forged
	// given patience.
	if !hmac.Equal([]byte(signature), []byte(want)) {
		return "", false
	}
	return token, true
}

// lookup resolves a verified token to its collector.
//
// One statement. It returns nothing for a session that is absent, expired, past the absolute cap,
// or whose collector has been removed — and "nothing" is refused as unauthenticated, which is
// FR-014 without a separate branch per condition.
func (s *SessionResolver) lookup(ctx context.Context, token string) (CollectorID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
		SELECT c.id
		  FROM "session" s
		  JOIN "user" u ON u."id" = s."userId"
		  JOIN collectors c ON c.user_id = u."id"
		 WHERE s."token" = $1
		   AND s."expiresAt" > now()
		   AND s."createdAt" > now() - make_interval(secs => $2)`,
		token, sessionAbsoluteCap.Seconds()).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrUnresolved
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("verify session: %w", err)
	}
	return id, nil
}
