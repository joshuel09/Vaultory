//go:build production

// Production counterpart to dev.go.
//
// The development resolver mints a session for anyone who asks, with no authentication. A build
// tag rather than an environment variable decides which of the two files is compiled, so in a
// production binary the signing key, the cookie issuer, and the seeded collector identifiers do
// not exist at all. A misconfigured VAULTORY_DEV_IDENTITY cannot turn authentication off here,
// because there is nothing left to turn on (FR-019).
package identity

import (
	"errors"
	"net/http"
)

// ErrDevIdentityUnavailable is returned by NewDevResolver in a production build. main refuses to
// start on it rather than continuing without a resolver.
var ErrDevIdentityUnavailable = errors.New(
	"development identity is not available in a production build: unset VAULTORY_DEV_IDENTITY, " +
		"or build without -tags production for local development")

// DevResolver exists only so that code holding a *DevResolver still compiles. A production build
// never constructs one, so every method below is unreachable.
type DevResolver struct{}

func NewDevResolver(string) (*DevResolver, error) {
	return nil, ErrDevIdentityUnavailable
}

func (d *DevResolver) Resolve(*http.Request) (CollectorID, error) {
	return CollectorID{}, ErrUnresolved
}

func (d *DevResolver) IssueSession(http.ResponseWriter, CollectorID) {}

func CollectorFor(string) CollectorID { return CollectorID{} }
