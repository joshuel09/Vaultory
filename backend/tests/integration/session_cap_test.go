//go:build integration

package integration

import "testing"

// FR-010's second half, and the case Better Auth alone would let through.
//
// Better Auth extends expiresAt as a session is used, so an actively used session never expires by
// that measure. The 90-day ceiling exists to stop one living indefinitely on a device its owner no
// longer controls, and Go is the only thing that enforces it. A healthy expiresAt paired with an
// old createdAt is precisely the combination that proves the cap is applied.
func TestTheAbsoluteCapIsEnforcedIndependentlyOfExpiry(t *testing.T) {
	freshStore(t)
	user, _ := userFor(t)

	insertSession(t, user, "cap-fresh", "20 days", "89 days")
	insertSession(t, user, "cap-exceeded", "20 days", "91 days")

	if _, err := resolve(t, sign("cap-fresh", testSecret)); err != nil {
		t.Fatalf("a session 89 days old with 20 days left was refused: %v", err)
	}
	if _, err := resolve(t, sign("cap-exceeded", testSecret)); err == nil {
		t.Fatal("a session 91 days old was accepted because its expiresAt looked healthy; " +
			"the absolute cap is not being applied")
	}
}
