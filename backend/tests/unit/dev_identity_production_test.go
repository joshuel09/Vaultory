//go:build production

package unit

import (
	"errors"
	"testing"

	"github.com/joshuel09/vaultory/backend/internal/identity"
)

// FR-019, checked rather than asserted.
//
// The development resolver mints a session for anyone who asks. This test is the executable form
// of the claim that a production binary cannot do that: run it with `go test -tags production`
// and it fails the moment someone reintroduces the resolver into a production build.
func TestDevResolverIsUnavailableInAProductionBuild(t *testing.T) {
	r, err := identity.NewDevResolver("test-secret")
	if err == nil {
		t.Fatal("production build constructed a development resolver: it would mint sessions " +
			"without authentication")
	}
	if !errors.Is(err, identity.ErrDevIdentityUnavailable) {
		t.Fatalf("unexpected error: %v", err)
	}
	if r != nil {
		t.Fatal("production build returned a non-nil resolver alongside its error")
	}
}
