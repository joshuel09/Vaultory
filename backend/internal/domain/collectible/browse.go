package collectible

import (
	"strings"
	"unicode/utf8"
)

// Browsing a vault: searching it and ordering it (feature 011).

// SortOrder is one of the ways a gallery can be ordered (FR-006).
//
// A closed set, like the statuses. The store turns each into SQL from a whitelist; nothing a
// collector sends is ever placed in a query as an order.
type SortOrder string

const (
	SortAddedDesc     SortOrder = "added_desc"
	SortAddedAsc      SortOrder = "added_asc"
	SortNameAsc       SortOrder = "name_asc"
	SortNameDesc      SortOrder = "name_desc"
	SortPriceDesc     SortOrder = "price_desc"
	SortPriceAsc      SortOrder = "price_asc"
	SortPurchasedDesc SortOrder = "purchased_desc"
	SortPurchasedAsc  SortOrder = "purchased_asc"
	SortReleasedDesc  SortOrder = "released_desc"
	SortReleasedAsc   SortOrder = "released_asc"
)

// DefaultSortOrder is newest added first, as the gallery has always been (001 FR-034).
const DefaultSortOrder = SortAddedDesc

var sortOrders = []SortOrder{
	SortAddedDesc, SortAddedAsc,
	SortNameAsc, SortNameDesc,
	SortPriceDesc, SortPriceAsc,
	SortPurchasedDesc, SortPurchasedAsc,
	SortReleasedDesc, SortReleasedAsc,
}

// AllSortOrders returns the permitted orders.
func AllSortOrders() []SortOrder {
	out := make([]SortOrder, len(sortOrders))
	copy(out, sortOrders)
	return out
}

// ParseSortOrder reads an order from a request. Empty means the default. Anything else unknown is
// refused rather than quietly ignored, so a mistyped link says so instead of showing the wrong order
// (FR-013).
func ParseSortOrder(raw string) (SortOrder, *Violation) {
	if raw == "" {
		return DefaultSortOrder, nil
	}
	for _, o := range sortOrders {
		if string(o) == raw {
			return o, nil
		}
	}
	return "", &Violation{
		Field:   "sort",
		Code:    CodeInvalidValue,
		Message: "That is not a way the collection can be ordered.",
	}
}

// Search limits (FR-004). The word limit bounds the work one search can ask for; nobody types twenty
// separate words to find one figure.
const (
	MaxSearchLength = 200
	MaxSearchWords  = 20
)

// ParseSearch turns what a collector typed into the words a search must find (FR-002, FR-004).
//
// Empty or whitespace-only is no search, and returns no words. Each word is kept exactly as typed —
// matching is case-insensitive and literal, but that is the store's job, because how to make a
// character literal depends on how it is matched (research.md Decision 1).
func ParseSearch(raw string) ([]string, *Violation) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(trimmed) > MaxSearchLength {
		return nil, &Violation{
			Field:   "q",
			Code:    CodeTooLong,
			Message: "Search for at most 200 characters.",
		}
	}
	words := strings.Fields(trimmed)
	if len(words) > MaxSearchWords {
		return nil, &Violation{
			Field:   "q",
			Code:    CodeInvalidValue,
			Message: "Search for at most 20 words.",
		}
	}
	return words, nil
}
