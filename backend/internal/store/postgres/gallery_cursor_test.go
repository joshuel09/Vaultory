package postgres

import (
	"encoding/base64"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
)

// T022 — FR-016: a page reference carrying a value its order's cast would reject is refused before
// the query, so it answers 400 rather than reaching the database and answering 500 (#71).

// rawCursor builds a v2 cursor by hand, as a tampered or truncated link would carry one.
func rawCursor(order collectible.SortOrder, missing, value string) string {
	raw := galleryCursorVersion + "|" + string(order) + "|" + missing + "|" + uuid.NewString() + "|" + value
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func TestGalleryCursorValues(t *testing.T) {
	cases := []struct {
		order collectible.SortOrder
		value string
		valid bool
	}{
		// Each cast's own text forms, as PostgreSQL writes them.
		{collectible.SortPriceDesc, "12.50", true},
		{collectible.SortPriceAsc, "0", true},
		{collectible.SortPriceAsc, "1234567890.00", true},
		{collectible.SortPurchasedAsc, "2026-10-27", true},
		{collectible.SortReleasedDesc, "1999-02-28", true},
		{collectible.SortAddedDesc, "2026-10-27T09:46:33.123456+00:00", true},
		{collectible.SortAddedDesc, "1880-06-01T15:21:10+05:21:10", true},
		{collectible.SortAddedDesc, "2026-10-27 09:46:33.123456+00", true},
		{collectible.SortAddedAsc, "2026-10-27 18:46:33+09", true},
		{collectible.SortAddedAsc, "2026-10-27 15:16:33.5+05:30", true},
		{collectible.SortNameAsc, "anything at all | even the separator", true},
		{collectible.SortNameDesc, "", true},

		// The issue's three, and their neighbours.
		{collectible.SortPriceDesc, "abc", false},
		{collectible.SortPriceDesc, "", false},
		{collectible.SortPriceDesc, "1e10", false},
		{collectible.SortPriceAsc, "12.50; DROP TABLE collectibles", false},
		{collectible.SortPurchasedAsc, "not-a-date", false},
		{collectible.SortPurchasedAsc, "2026-02-30", false},
		{collectible.SortReleasedAsc, "2026-10-27 09:46:33+00", false},
		{collectible.SortAddedDesc, "nonsense", false},
		{collectible.SortAddedDesc, "2026-10-27", false},
		{collectible.SortAddedAsc, "2026-13-27 09:46:33+00", false},
		// What a non-ISO DateStyle would have rendered — never issued since to_json (#85).
		{collectible.SortPurchasedAsc, "06/11/2026", false},
		{collectible.SortAddedDesc, "06/11/2026 03:50:24.630137 UTC", false},
	}
	for _, c := range cases {
		_, err := decodeGalleryCursor(rawCursor(c.order, "0", c.value), c.order)
		if c.valid && err != nil {
			t.Errorf("%s %q: refused (%v), want accepted", c.order, c.value, err)
		}
		if !c.valid && !errors.Is(err, ErrCursorInvalid) {
			t.Errorf("%s %q: got %v, want ErrCursorInvalid", c.order, c.value, err)
		}
	}
}

// When the last entry was missing the value, the value is never bound, so it is not checked.
func TestGalleryCursorMissingValueIsNotChecked(t *testing.T) {
	if _, err := decodeGalleryCursor(rawCursor(collectible.SortPriceDesc, "1", ""), collectible.SortPriceDesc); err != nil {
		t.Errorf("a missing-value cursor was refused: %v", err)
	}
}

// Every order's own cursor survives the check it is now put through.
func TestGalleryCursorRoundTrip(t *testing.T) {
	values := map[string]string{
		"timestamptz": "2026-10-27T09:46:33.123456+00:00",
		"text":        "aerith",
		"numeric":     "42.00",
		"date":        "2026-10-27",
	}
	for order, spec := range sortSpecs {
		in := galleryCursor{order: order, value: values[spec.cast], id: uuid.New()}
		out, err := decodeGalleryCursor(encodeGalleryCursor(in), order)
		if err != nil {
			t.Errorf("%s: own cursor refused: %v", order, err)
			continue
		}
		if out != in {
			t.Errorf("%s: round trip gave %+v, want %+v", order, out, in)
		}
	}
}
