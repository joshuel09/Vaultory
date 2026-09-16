//go:build !production

package unit

import (
	"testing"

	"github.com/joshuel09/vaultory/backend/internal/identity"
)

// The development build is the one developers run. It must have a working resolver, or nobody can
// sign in locally at all.
func TestDevResolverIsAvailableInADevelopmentBuild(t *testing.T) {
	r, err := identity.NewDevResolver("test-secret")
	if err != nil {
		t.Fatalf("development build refused to construct the dev resolver: %v", err)
	}
	if r == nil {
		t.Fatal("development build returned a nil resolver and no error")
	}
}
