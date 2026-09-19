package unit

import (
	"strings"
	"testing"

	"github.com/joshuel09/vaultory/backend/internal/config"
)

// FR-019a, SC-014.
//
// The session secret is the only input to the signature the resolver verifies. A short one makes
// every session forgeable, and — unlike a wrong password or a missing database — nothing about the
// running system would look wrong. Refusing to start is the entire protection, so it is worth a
// test rather than a line in the README.
func TestStartupRefusesAWeakSessionSecret(t *testing.T) {
	const valid = "0123456789abcdef0123456789abcdef" // exactly 32

	cases := []struct {
		name   string
		secret string
		ok     bool
	}{
		{"absent", "", false},
		{"one character", "x", false},
		{"one short of the minimum", strings.Repeat("a", config.MinSessionSecretLength-1), false},
		{"exactly the minimum", valid, true},
		{"comfortably over", strings.Repeat("b", 64), true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("VAULTORY_DATABASE_URL", "postgres://localhost/x")
			t.Setenv("VAULTORY_IMAGE_STORE", "filesystem")
			t.Setenv("VAULTORY_IMAGE_STORE_PATH", t.TempDir())
			t.Setenv("VAULTORY_SESSION_SECRET", c.secret)

			_, err := config.Load()
			switch {
			case c.ok && err != nil:
				t.Fatalf("a %d-character secret was refused: %v", len(c.secret), err)
			case !c.ok && err == nil:
				t.Fatalf("a %d-character secret was accepted; every session it signs is forgeable",
					len(c.secret))
			case !c.ok && !strings.Contains(err.Error(), "VAULTORY_SESSION_SECRET"):
				t.Fatalf("the refusal does not name the variable at fault: %v", err)
			}
		})
	}
}

// The minimum is referenced by compose files and the README, so it is worth pinning: changing it
// silently would leave those out of step.
func TestSessionSecretMinimumIs32(t *testing.T) {
	if config.MinSessionSecretLength != 32 {
		t.Errorf("minimum is %d; compose.yaml, compose.prod.yaml and README.md assume 32",
			config.MinSessionSecretLength)
	}
}
