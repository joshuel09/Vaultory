//go:build !production

// This file is excluded from a production build by the tag above. The counterpart,
// dev_production.go, replaces it with a constructor that refuses. FR-019 is therefore satisfied by
// what is compiled in, not by a runtime flag that a misconfiguration could get wrong.
package identity

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SessionCookieName is the development session cookie.
const SessionCookieName = "vaultory_dev_session"

// Seeded development collectors, matching migration 000005. Two, not one, because a single
// collector cannot demonstrate cross-collector isolation (FR-026, FR-027).
var (
	DevCollectorPrimary = uuid.MustParse("11111111-1111-4111-8111-111111111111")
	DevCollectorSecond  = uuid.MustParse("22222222-2222-4222-8222-222222222222")
)

// DevResolver resolves the acting collector from a signed cookie this package itself issues.
//
// This exists only because authentication is out of scope while collector-private data is
// required. It is gated on VAULTORY_DEV_IDENTITY=enabled and must never run outside development:
// it mints a session for anyone who asks, without any authentication whatsoever.
type DevResolver struct {
	secret []byte
}

// NewDevResolver builds the development resolver. The error return exists for the production
// build's counterpart, which always fails; here it is always nil.
func NewDevResolver(secret string) (*DevResolver, error) {
	return &DevResolver{secret: []byte(secret)}, nil
}

// Resolve reads the signed cookie. Note what it does not do: it never consults a header, a query
// parameter, or a request body for an identity (FR-028).
func (d *DevResolver) Resolve(r *http.Request) (CollectorID, error) {
	c, err := r.Cookie(SessionCookieName)
	if err != nil || c.Value == "" {
		return uuid.Nil, ErrUnresolved
	}
	rawID, sig, ok := strings.Cut(c.Value, ".")
	if !ok {
		return uuid.Nil, ErrUnresolved
	}
	// Constant-time comparison: a signature check that leaks timing is a signature check that can
	// be forged given patience.
	if !hmac.Equal([]byte(sig), []byte(d.sign(rawID))) {
		return uuid.Nil, ErrUnresolved
	}
	id, err := uuid.Parse(rawID)
	if err != nil {
		return uuid.Nil, ErrUnresolved
	}
	return id, nil
}

// IssueSession sets the development session cookie for the given collector.
func (d *DevResolver) IssueSession(w http.ResponseWriter, id CollectorID) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    fmt.Sprintf("%s.%s", id.String(), d.sign(id.String())),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})
}

func (d *DevResolver) sign(payload string) string {
	mac := hmac.New(sha256.New, d.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// CollectorFor maps the dev sign-in's optional collector selector to a seeded collector.
// "second" selects the second collector, so the privacy walkthroughs in quickstart.md can be run
// from a browser or curl rather than only from the integration suite.
func CollectorFor(selector string) CollectorID {
	if selector == "second" {
		return DevCollectorSecond
	}
	return DevCollectorPrimary
}
