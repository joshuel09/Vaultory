package unit

import (
	"testing"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
)

func TestParseCollectionStatusAcceptsExactlyFour(t *testing.T) {
	for _, want := range []string{"owned", "preordered", "wishlist", "sold"} {
		got, err := collectible.ParseCollectionStatus(want)
		if err != nil {
			t.Fatalf("ParseCollectionStatus(%q) returned error: %v", want, err)
		}
		if got.String() != want {
			t.Errorf("ParseCollectionStatus(%q) = %q", want, got)
		}
	}
	if n := len(collectible.AllStatuses()); n != 4 {
		t.Errorf("AllStatuses() has %d entries, want exactly 4 (FR-004)", n)
	}
}

// FR-004: any other value MUST be rejected. Casing and padding are not coerced, because a status
// is chosen from a fixed set rather than typed.
func TestParseCollectionStatusRejectsEverythingElse(t *testing.T) {
	rejected := []string{
		"", "borrowed", "traded", "OWNED", "Owned", " owned", "owned ",
		"own", "ownedd", "wish list", "null", "0",
	}
	for _, raw := range rejected {
		if _, err := collectible.ParseCollectionStatus(raw); err == nil {
			t.Errorf("ParseCollectionStatus(%q) was accepted; FR-004 requires rejection", raw)
		}
	}
}

func TestCollectionStatusValid(t *testing.T) {
	if !collectible.StatusOwned.Valid() {
		t.Error("StatusOwned should be valid")
	}
	if collectible.CollectionStatus("borrowed").Valid() {
		t.Error("an unknown status should not be valid")
	}
}
