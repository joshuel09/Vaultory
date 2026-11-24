package postgres

import (
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
)

// The gallery's page reference (feature 011, research.md Decision 2).
//
// It carries everything the next page's predicate needs: which order it belongs to, whether the
// last entry was missing the sorted-by value, that value, and the last entry's id. The value is the
// key as the database renders it with to_json — ISO 8601 for dates and timestamps, whatever the
// session's DateStyle (FR-017) — and goes back with an explicit cast, so a price is never a float on
// the way round (Constitution IV) and a tie stays a tie.
//
// Opaque to the client. Its layout is an encoding detail, not part of the contract.

// galleryCursorVersion marks this layout, so a legacy two-part cursor can still be told apart.
const galleryCursorVersion = "v2"

// ErrCursorInvalid is a page reference this system did not issue, or one issued for another order.
// It is the client's mistake, so the transport answers it with 400 (feature 011, FR-012, FR-016).
// Shared by the gallery and the Trash, which test for it with errors.Is rather than by its text.
var ErrCursorInvalid = errors.New("cursor is not valid")

type galleryCursor struct {
	order   collectible.SortOrder
	missing bool
	value   string // the key as text; meaningless when missing
	id      uuid.UUID
}

func encodeGalleryCursor(c galleryCursor) string {
	missing := "0"
	if c.missing {
		missing = "1"
	}
	// The value goes last, because a name can contain anything — including the separator.
	raw := strings.Join([]string{galleryCursorVersion, string(c.order), missing, c.id.String(), c.value}, "|")
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// decodeGalleryCursor reads a cursor for the given order. A cursor issued for another order is
// refused rather than reinterpreted: applying one order's position to another produces a page that
// silently skips or repeats (FR-012).
func decodeGalleryCursor(raw string, order collectible.SortOrder) (galleryCursor, error) {
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return galleryCursor{}, ErrCursorInvalid
	}
	parts := strings.SplitN(string(data), "|", 5)

	// Legacy: nanos|id, issued before feature 011, always newest added first. Still read, so a
	// gallery open across a deploy can keep scrolling.
	if len(parts) == 2 {
		if order != collectible.SortAddedDesc {
			return galleryCursor{}, ErrCursorInvalid
		}
		nanos, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return galleryCursor{}, ErrCursorInvalid
		}
		id, err := uuid.Parse(parts[1])
		if err != nil {
			return galleryCursor{}, ErrCursorInvalid
		}
		return galleryCursor{
			order: order,
			value: time.Unix(0, nanos).UTC().Format(time.RFC3339Nano),
			id:    id,
		}, nil
	}

	if len(parts) != 5 || parts[0] != galleryCursorVersion || parts[1] != string(order) {
		return galleryCursor{}, ErrCursorInvalid
	}
	if parts[2] != "0" && parts[2] != "1" {
		return galleryCursor{}, ErrCursorInvalid
	}
	id, err := uuid.Parse(parts[3])
	if err != nil {
		return galleryCursor{}, ErrCursorInvalid
	}
	c := galleryCursor{order: order, missing: parts[2] == "1", id: id, value: parts[4]}
	// The value is cast back in the query, and a cast that fails is a database error, not a
	// refusal: a reference altered to carry "abc" as a price answered 500 (#71). So nothing goes
	// to the cast that the cast could reject. When the last entry was missing the value, it is
	// not sent at all.
	if !c.missing {
		spec, err := specFor(order)
		if err != nil || !validCursorValue(spec.cast, c.value) {
			return galleryCursor{}, ErrCursorInvalid
		}
	}
	return c, nil
}

// cursorNumeric is numeric's text form as PostgreSQL writes it for a price: digits, and a fraction.
var cursorNumeric = regexp.MustCompile(`^-?[0-9]{1,20}(\.[0-9]{1,20})?$`)

// cursorTimestampLayouts are the timestamp forms the cursor carries. The store renders the key with
// to_json, which is ISO 8601 ("T" between date and time) whatever the session's DateStyle, so this
// check does not depend on how the database is configured to display dates (FR-017, #85). The
// offset is hours, hours and minutes, or all three — that one follows the session's TimeZone. The
// space-separated forms are what ::text wrote before #85, still accepted so a reference issued
// then keeps working.
var cursorTimestampLayouts = []string{
	"2006-01-02T15:04:05.999999999-07",
	"2006-01-02T15:04:05.999999999-07:00",
	"2006-01-02T15:04:05.999999999-07:00:00",
	"2006-01-02 15:04:05.999999999-07",
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999-07:00:00",
}

// validCursorValue says whether value is something its cast accepts — the sorted-by key as the store
// rendered it with to_json, and encodeGalleryCursor wrote it.
func validCursorValue(cast, value string) bool {
	switch cast {
	case "text":
		return true
	case "numeric":
		return cursorNumeric.MatchString(value)
	case "date":
		_, err := time.Parse("2006-01-02", value)
		return err == nil
	case "timestamptz":
		for _, layout := range cursorTimestampLayouts {
			if _, err := time.Parse(layout, value); err == nil {
				return true
			}
		}
		return false
	}
	return false
}

// sortSpec is how one order becomes SQL. Every field is a fixed string chosen by the order; nothing
// from a request reaches the query text (Constitution IV).
type sortSpec struct {
	key  string // the expression ordered by
	cast string // the type its text form is cast back to in the cursor predicate
	desc bool
}

var sortSpecs = map[collectible.SortOrder]sortSpec{
	collectible.SortAddedDesc:     {key: "c.created_at", cast: "timestamptz", desc: true},
	collectible.SortAddedAsc:      {key: "c.created_at", cast: "timestamptz", desc: false},
	collectible.SortNameAsc:       {key: "lower(c.name)", cast: "text", desc: false},
	collectible.SortNameDesc:      {key: "lower(c.name)", cast: "text", desc: true},
	collectible.SortPriceDesc:     {key: "c.purchase_price", cast: "numeric", desc: true},
	collectible.SortPriceAsc:      {key: "c.purchase_price", cast: "numeric", desc: false},
	collectible.SortPurchasedDesc: {key: "c.purchase_date", cast: "date", desc: true},
	collectible.SortPurchasedAsc:  {key: "c.purchase_date", cast: "date", desc: false},
	collectible.SortReleasedDesc:  {key: "c.release_date", cast: "date", desc: true},
	collectible.SortReleasedAsc:   {key: "c.release_date", cast: "date", desc: false},
}

func specFor(order collectible.SortOrder) (sortSpec, error) {
	if order == "" {
		order = collectible.DefaultSortOrder
	}
	spec, ok := sortSpecs[order]
	if !ok {
		return sortSpec{}, fmt.Errorf("unknown sort order %q", order)
	}
	return spec, nil
}

// orderBy puts entries missing the key last in either direction, then the key, then the id in the
// same direction so the order is total (FR-007, FR-008).
func (s sortSpec) orderBy() string {
	dir := "ASC"
	if s.desc {
		dir = "DESC"
	}
	return fmt.Sprintf("(%[1]s) IS NULL, %[1]s %[2]s, c.id %[2]s", s.key, dir)
}

// after is the predicate for "comes after the cursor in this order", using the three placeholders
// given for the cursor's value, id and missing flag. Spelled out because one row comparison cannot
// put missing values last in both directions (research.md Decision 2).
func (s sortSpec) after(value, id, missing string) string {
	op := ">"
	if s.desc {
		op = "<"
	}
	k := "(" + s.key + ")"
	v := value + "::" + s.cast
	return fmt.Sprintf(`(
		(NOT %[5]s AND ((%[1]s IS NOT NULL AND (%[1]s %[2]s %[3]s OR (%[1]s = %[3]s AND c.id %[2]s %[4]s))) OR %[1]s IS NULL))
		OR (%[5]s AND %[1]s IS NULL AND c.id %[2]s %[4]s)
	)`, k, op, v, id, missing)
}

// likeEscape makes every character of a search word literal under ILIKE, whose default escape
// character is the backslash (FR-003). It lives here, not in the domain, because it is a detail of
// how the store matches (research.md Decision 1).
func likeEscape(word string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(word)
}
