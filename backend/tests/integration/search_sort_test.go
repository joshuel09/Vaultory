//go:build integration

package integration

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/collection"
	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
)

// Feature 011: searching a vault, and ordering it.

func search(t *testing.T, svc *collection.Service, collector uuid.UUID, raw string, extra postgres.ListFilter) []string {
	t.Helper()
	words, v := collectible.ParseSearch(raw)
	if v != nil {
		t.Fatalf("parse %q: %+v", raw, v)
	}
	extra.Search = words
	page, err := svc.List(context.Background(), collector, extra, "", collection.MaxPageLimit)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var names []string
	for _, r := range page.Rows {
		names = append(names, r.Collectible.Name)
	}
	return names
}

func addDetailed(t *testing.T, svc *collection.Service, collector uuid.UUID, key string, set func(*collectible.Draft)) postgres.Row {
	t.Helper()
	d := draft(key, key)
	set(&d)
	added, v, err := svc.Add(context.Background(), collector, d)
	if err != nil || len(v) > 0 {
		t.Fatalf("add %s: %v %+v", key, err, v)
	}
	return added.Row
}

// T007 — FR-002: each searchable attribute matches, case-insensitively, as a substring.
func TestSearchMatchesEveryAttribute(t *testing.T) {
	_, svc := freshStore(t)
	cases := map[string]func(*collectible.Draft){
		"by-name":         func(d *collectible.Draft) { d.Name = "Kaiju Sentinel" },
		"by-character":    func(d *collectible.Draft) { d.Character = strPtr("Ultraman Tiga") },
		"by-series":       func(d *collectible.Draft) { d.Series = strPtr("Gundam Wing") },
		"by-manufacturer": func(d *collectible.Draft) { d.Manufacturer = strPtr("Prime 1 Studio") },
		"by-category":     func(d *collectible.Draft) { d.Category = strPtr("Statue") },
		"by-scale":        func(d *collectible.Draft) { d.Scale = strPtr("1/4") },
		"by-edition":      func(d *collectible.Draft) { d.Edition = strPtr("Exclusive Bonus") },
		"by-notes":        func(d *collectible.Draft) { d.Notes = strPtr("Signed by the sculptor") },
	}
	for key, set := range cases {
		addDetailed(t, svc, collectorA, key, set)
	}
	for term, want := range map[string]string{
		"sentinel": "Kaiju Sentinel", // name, part-word, different case
		"TIGA":     "by-character",
		"wing":     "by-series",
		"prime 1":  "by-manufacturer",
		"statu":    "by-category",
		"1/4":      "by-scale",
		"bonus":    "by-edition",
		"sculpt":   "by-notes",
	} {
		got := search(t, svc, collectorA, term, postgres.ListFilter{})
		if len(got) != 1 || got[0] != want {
			t.Errorf("search %q = %v, want [%s]", term, got, want)
		}
	}
}

// T007 — FR-002: several words must each appear, in any attributes, but not straddle two.
func TestSearchWordsCombineAcrossAttributes(t *testing.T) {
	_, svc := freshStore(t)
	addDetailed(t, svc, collectorA, "Kaiju Sentinel", func(d *collectible.Draft) {
		d.Manufacturer = strPtr("Prime 1")
		d.Series = strPtr("Kaiju Wars")
	})
	addDetailed(t, svc, collectorA, "Kaiju Brute", func(d *collectible.Draft) { d.Manufacturer = strPtr("Hot Toys") })

	if got := search(t, svc, collectorA, "kaiju prime", postgres.ListFilter{}); len(got) != 1 || got[0] != "Kaiju Sentinel" {
		t.Errorf("words across attributes: %v, want just Kaiju Sentinel", got)
	}
	if got := search(t, svc, collectorA, "kaiju", postgres.ListFilter{}); len(got) != 2 {
		t.Errorf("one word in both: %v", got)
	}
	// "sentinelprime" spans the end of the name and the start of the manufacturer. The attributes
	// are joined with a newline, so it must not match.
	if got := search(t, svc, collectorA, "sentinelprime", postgres.ListFilter{}); len(got) != 0 {
		t.Errorf("a match straddled two attributes: %v", got)
	}
}

// T007 — FR-003: %, _ and \ are literal.
func TestSearchCharactersAreLiteral(t *testing.T) {
	_, svc := freshStore(t)
	addDetailed(t, svc, collectorA, "100% Complete", func(*collectible.Draft) {})
	addDetailed(t, svc, collectorA, "a_b Model", func(*collectible.Draft) {})
	addDetailed(t, svc, collectorA, `Back\slash`, func(*collectible.Draft) {})
	addDetailed(t, svc, collectorA, "Plain Figure", func(*collectible.Draft) {})

	for term, want := range map[string]int{
		"%":   1, // a wildcard would match all four
		"_":   1, // a wildcard would match all four
		`\`:   1,
		"0%":  1,
		"a_b": 1,
		"axb": 0, // _ as a wildcard would match "a_b"... and nothing else; literal matches nothing
	} {
		if got := search(t, svc, collectorA, term, postgres.ListFilter{}); len(got) != want {
			t.Errorf("search %q matched %v, want %d", term, got, want)
		}
	}
}

// T007 — FR-014, SC-006: neither another collector's vault nor the Trash is searched.
func TestSearchStaysInTheCollectorsOwnVault(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()
	addDetailed(t, svc, collectorB, "Kaiju Theirs", func(*collectible.Draft) {})
	trashed := addDetailed(t, svc, collectorA, "Kaiju Trashed", func(*collectible.Draft) {})
	addDetailed(t, svc, collectorA, "Kaiju Mine", func(*collectible.Draft) {})
	if err := svc.Delete(ctx, collectorA, trashed.Collectible.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if got := search(t, svc, collectorA, "kaiju", postgres.ListFilter{}); len(got) != 1 || got[0] != "Kaiju Mine" {
		t.Errorf("search reached beyond the collector's vault: %v", got)
	}
}

// seedForSorting inserts n collectibles straight into the table, with heavy ties and missing values
// in every sorted attribute — the cases that break paging.
func seedForSorting(t *testing.T, collector uuid.UUID, n int) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO collectibles (collector_id, name, collection_status, purchase_price,
		                          purchase_date, release_date, created_at)
		SELECT $1,
		       -- Five names in mixed case, so name order has ties and must ignore case.
		       (ARRAY['alpha', 'Alpha', 'beta', 'Gamma', 'gamma'])[1 + g % 5] || '',
		       'owned',
		       CASE WHEN g % 3 = 0 THEN NULL ELSE ((g % 7) * 100 + 0.5)::numeric(12,2) END,
		       CASE WHEN g % 4 = 0 THEN NULL ELSE DATE '2026-01-01' + (g % 11) END,
		       CASE WHEN g % 5 = 0 THEN NULL ELSE DATE '2027-01-01' + (g % 13) END,
		       -- Blocks of ten share a created_at, so even the default order has ties.
		       TIMESTAMPTZ '2026-09-01 00:00:00+00' + make_interval(mins => g / 10)
		FROM generate_series(1, $2) AS g`, collector, n)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
}

// T010 — FR-007, FR-008, SC-002, SC-003, SC-005: every order pages exactly once, in order, with
// missing values last, over 500 collectibles in pages of 24.
func TestEveryOrderPagesExactlyOnce(t *testing.T) {
	store, svc := freshStore(t)
	ctx := context.Background()
	const n = 500
	seedForSorting(t, collectorA, n)

	for _, order := range collectible.AllSortOrders() {
		t.Run(string(order), func(t *testing.T) {
			filter := postgres.ListFilter{Sort: order}

			// The whole order in one query, as the reference.
			whole, err := store.List(ctx, collectorA, filter, "", 1000)
			if err != nil {
				t.Fatalf("whole: %v", err)
			}
			if len(whole.Rows) != n {
				t.Fatalf("whole order holds %d, want %d", len(whole.Rows), n)
			}

			// The same order, 24 at a time, through the service as the API uses it.
			var paged []postgres.Row
			cursor := ""
			start := time.Now()
			var firstPage time.Duration
			for pages := 0; ; pages++ {
				page, err := svc.List(ctx, collectorA, filter, cursor, 24)
				if err != nil {
					t.Fatalf("page %d: %v", pages, err)
				}
				if pages == 0 {
					firstPage = time.Since(start)
				}
				paged = append(paged, page.Rows...)
				if page.NextCursor == "" {
					break
				}
				cursor = page.NextCursor
				if pages > n {
					t.Fatal("paging does not terminate")
				}
			}
			t.Logf("first page in %v", firstPage)

			if len(paged) != n {
				t.Fatalf("paged through %d, want %d", len(paged), n)
			}
			seen := map[uuid.UUID]bool{}
			for i, r := range paged {
				if seen[r.Collectible.ID] {
					t.Fatalf("position %d repeats %v", i, r.Collectible.ID)
				}
				seen[r.Collectible.ID] = true
				if r.Collectible.ID != whole.Rows[i].Collectible.ID {
					t.Fatalf("position %d differs from the order in one query", i)
				}
			}
			checkOrder(t, order, paged)
		})
	}
}

// checkOrder verifies the order independently of the SQL that produced it, for the attributes Go
// can compare without a collation: missing values last, then values in the requested direction.
func checkOrder(t *testing.T, order collectible.SortOrder, rows []postgres.Row) {
	t.Helper()
	desc := strings.HasSuffix(string(order), "_desc")
	key := func(r postgres.Row) (string, bool) {
		c := r.Collectible
		switch {
		case strings.HasPrefix(string(order), "price"):
			if c.PurchasePrice == nil {
				return "", false
			}
			// Fixed width so string order is numeric order. A float is fine here: this only orders
			// the fixture's small amounts in a test, it never computes money.
			f, err := strconv.ParseFloat(c.PurchasePrice.String(), 64)
			if err != nil {
				t.Fatalf("price %q: %v", c.PurchasePrice.String(), err)
			}
			return fmt.Sprintf("%015.2f", f), true
		case strings.HasPrefix(string(order), "purchased"):
			if c.PurchaseDate == nil {
				return "", false
			}
			return c.PurchaseDate.Format("2006-01-02"), true
		case strings.HasPrefix(string(order), "released"):
			if c.ReleaseDate == nil {
				return "", false
			}
			return c.ReleaseDate.Format("2006-01-02"), true
		case strings.HasPrefix(string(order), "added"):
			return c.CreatedAt.UTC().Format(time.RFC3339Nano), true
		default:
			return strings.ToLower(c.Name), true // the fixture's names are plain ASCII
		}
	}
	seenMissing := false
	prev, havePrev := "", false
	for i, r := range rows {
		k, ok := key(r)
		if !ok {
			seenMissing = true
			continue
		}
		if seenMissing {
			t.Fatalf("position %d has a value after a missing one — missing values must come last (FR-007)", i)
		}
		if havePrev && ((desc && k > prev) || (!desc && k < prev)) {
			t.Fatalf("position %d (%s) is out of order after %s", i, k, prev)
		}
		prev, havePrev = k, true
	}
}

// T011 — FR-012: a cursor belongs to its order; the legacy cursor still pages newest-first.
func TestACursorBelongsToItsOrder(t *testing.T) {
	store, svc := freshStore(t)
	ctx := context.Background()
	seedForSorting(t, collectorA, 60)

	page, err := svc.List(ctx, collectorA, postgres.ListFilter{Sort: collectible.SortPriceDesc}, "", 24)
	if err != nil || page.NextCursor == "" {
		t.Fatalf("first page: %v", err)
	}
	if _, err := svc.List(ctx, collectorA, postgres.ListFilter{Sort: collectible.SortNameAsc}, page.NextCursor, 24); err == nil {
		t.Error("a price-order cursor was accepted for name order")
	}
	if _, err := svc.List(ctx, collectorA, postgres.ListFilter{}, page.NextCursor, 24); err == nil {
		t.Error("a price-order cursor was accepted for the default order")
	}

	// A cursor in the pre-feature-011 layout, as a gallery open across the deploy would hold.
	whole, err := store.List(ctx, collectorA, postgres.ListFilter{}, "", 1000)
	if err != nil {
		t.Fatalf("whole: %v", err)
	}
	last := whole.Rows[23].Collectible
	legacy := encodeLegacyCursor(last.CreatedAt, last.ID)
	next, err := svc.List(ctx, collectorA, postgres.ListFilter{}, legacy, 24)
	if err != nil {
		t.Fatalf("legacy cursor refused: %v", err)
	}
	if len(next.Rows) == 0 || next.Rows[0].Collectible.ID != whole.Rows[24].Collectible.ID {
		t.Error("a legacy cursor did not continue where it left off")
	}
	if _, err := svc.List(ctx, collectorA, postgres.ListFilter{Sort: collectible.SortNameAsc}, legacy, 24); err == nil {
		t.Error("a legacy cursor was accepted for a non-default order")
	}
}

// T012 — FR-010: filters, search and sort compose.
func TestSearchSortAndFiltersCompose(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()
	mk := func(key, status, series, price, notes string) {
		addDetailed(t, svc, collectorA, key, func(d *collectible.Draft) {
			d.Status = status
			d.Series = strPtr(series)
			d.PurchasePrice = strPtr(price)
			d.Notes = strPtr(notes)
		})
	}
	// Each of the last three passes every narrowing but one.
	mk("Kaiju A", "owned", "Kaiju Wars", "300.00", "boxed")
	mk("Kaiju B", "owned", "Kaiju Wars", "900.00", "boxed")
	mk("Kaiju C", "sold", "Kaiju Wars", "1500.00", "boxed") // wrong status
	mk("Kaiju D", "owned", "Other", "1200.00", "boxed")     // wrong series
	mk("Mecha E", "owned", "Kaiju Wars", "50.00", "loose")  // does not match the search

	owned := collectible.CollectionStatus("owned")
	series := "kaiju wars"
	words, _ := collectible.ParseSearch("boxed")
	page, err := svc.List(ctx, collectorA, postgres.ListFilter{
		Status: &owned, Series: &series, Search: words, Sort: collectible.SortPriceDesc,
	}, "", collection.MaxPageLimit)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var got []string
	for _, r := range page.Rows {
		got = append(got, r.Collectible.Name)
	}
	if strings.Join(got, ",") != "Kaiju B,Kaiju A" {
		t.Errorf("composed result = %v, want [Kaiju B Kaiju A]", got)
	}
	if page.TotalUnfiltered != 5 {
		t.Errorf("TotalUnfiltered = %d, want the whole vault, 5", page.TotalUnfiltered)
	}
}

// encodeLegacyCursor writes a cursor in the layout used before feature 011 — nanoseconds and id. The
// format is spelled out here, not borrowed from the store, because it is the promise under test: a
// cursor a browser already holds must keep working.
func encodeLegacyCursor(t time.Time, id uuid.UUID) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(t.UTC().UnixNano(), 10) + "|" + id.String()))
}
