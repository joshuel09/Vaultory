// Package collectible holds Vaultory's collectible domain: the entry a collector records in their
// vault, and the rules that govern it.
//
// Nothing in this package imports net/http, database/sql, or any driver. Business logic must be
// testable without a transport or a database (Constitution Principle II).
package collectible

import "fmt"

// CollectionStatus is the state of a collectible in a collector's vault.
//
// There are exactly four, and no others are accepted (FR-004). They are peers rather than stages:
// a collectible may be recorded directly as Sold without ever having been Owned.
type CollectionStatus string

const (
	StatusOwned      CollectionStatus = "owned"
	StatusPreordered CollectionStatus = "preordered"
	StatusWishlist   CollectionStatus = "wishlist"
	StatusSold       CollectionStatus = "sold"
)

// allStatuses is the closed set. Adding a status here is a specification change, not a code change.
var allStatuses = []CollectionStatus{StatusOwned, StatusPreordered, StatusWishlist, StatusSold}

// AllStatuses returns the permitted statuses, for callers that need to present or validate them.
func AllStatuses() []CollectionStatus {
	out := make([]CollectionStatus, len(allStatuses))
	copy(out, allStatuses)
	return out
}

// ParseCollectionStatus accepts only the four permitted values, exactly as written. It does not
// trim, lowercase, or otherwise coerce its input: a status is chosen from a fixed set rather than
// typed, so a value that does not match is a caller error, not a formatting one.
func ParseCollectionStatus(raw string) (CollectionStatus, error) {
	for _, s := range allStatuses {
		if string(s) == raw {
			return s, nil
		}
	}
	return "", fmt.Errorf("unknown collection status %q", raw)
}

// Valid reports whether s is one of the four permitted statuses.
func (s CollectionStatus) Valid() bool {
	for _, known := range allStatuses {
		if s == known {
			return true
		}
	}
	return false
}

func (s CollectionStatus) String() string { return string(s) }
