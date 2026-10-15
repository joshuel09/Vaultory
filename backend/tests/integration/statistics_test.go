//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
)

// priced builds a draft with a purchase price and the given status, series and manufacturer.
//
// Every field is explicit because the tests below are about which rows land in which group, and a
// helper that defaulted anything would hide the thing under test.
func priced(key, name, status, price, series, manufacturer string) collectible.Draft {
	d := collectible.Draft{
		SubmissionKey: key,
		Submitted:     collectible.Submitted{Name: name, Status: status},
	}
	if price != "" {
		d.PurchasePrice = &price
	}
	if series != "" {
		d.Series = &series
	}
	if manufacturer != "" {
		d.Manufacturer = &manufacturer
	}
	return d
}

func statusOf(t *testing.T, stats collectible.CollectionStatistics, s collectible.CollectionStatus) collectible.StatusSummary {
	t.Helper()
	for _, row := range stats.ByStatus {
		if row.Status == s {
			return row
		}
	}
	t.Fatalf("no summary for status %q; all four must always be present", s)
	return collectible.StatusSummary{}
}

func groupLabelled(groups []collectible.GroupSummary, label string) (collectible.GroupSummary, bool) {
	for _, g := range groups {
		if g.Label == label {
			return g, true
		}
	}
	return collectible.GroupSummary{}, false
}

func groupOfKind(groups []collectible.GroupSummary, kind collectible.GroupKind) (collectible.GroupSummary, bool) {
	for _, g := range groups {
		if g.Kind == kind {
			return g, true
		}
	}
	return collectible.GroupSummary{}, false
}

// T024: counts and totals over a vault of known contents.
func TestStatisticsOverAKnownVault(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	for _, d := range []collectible.Draft{
		priced("s-1", "Owned One", "owned", "100.00", "Fate", "Good Smile Company"),
		priced("s-2", "Owned Two", "owned", "250.50", "Fate", "Good Smile Company"),
		priced("s-3", "Preorder", "preordered", "80.00", "Monogatari", "Alter"),
		priced("s-4", "Sold One", "sold", "30.00", "Fate", "Alter"),
	} {
		if _, v, err := svc.Add(ctx, collectorA, d); err != nil || len(v) > 0 {
			t.Fatalf("add %q: %v %+v", d.Name, err, v)
		}
	}

	stats, err := svc.Statistics(ctx, collectorA)
	if err != nil {
		t.Fatalf("statistics: %v", err)
	}

	if stats.TotalCount != 4 {
		t.Errorf("totalCount = %d, want 4", stats.TotalCount)
	}
	if stats.PricedCount != 4 {
		t.Errorf("pricedCount = %d, want 4", stats.PricedCount)
	}
	if got := statusOf(t, stats, collectible.StatusOwned); got.Count != 2 || got.Total.String() != "350.50" {
		t.Errorf("owned = (%d, %q), want (2, \"350.50\")", got.Count, got.Total.String())
	}
	if got := statusOf(t, stats, collectible.StatusPreordered); got.Total.String() != "80.00" {
		t.Errorf("preordered total = %q, want \"80.00\"", got.Total.String())
	}
	if got := statusOf(t, stats, collectible.StatusSold); got.Total.String() != "30.00" {
		t.Errorf("sold total = %q, want \"30.00\"", got.Total.String())
	}
}

// T025: exactness, against a total assembled independently of the code under test.
//
// 0.10 + 0.20 + 0.30 is the canonical floating-point failure. The expectation is a literal rather
// than a computed value, so this test cannot agree with a bug by reproducing it (SC-002).
func TestStatisticsTotalsAreExact(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	for i, price := range []string{"0.10", "0.20", "0.30"} {
		d := priced(fmt.Sprintf("x-%d", i), fmt.Sprintf("Awkward %d", i), "owned", price, "", "")
		if _, v, err := svc.Add(ctx, collectorA, d); err != nil || len(v) > 0 {
			t.Fatalf("add %q: %v %+v", price, err, v)
		}
	}

	stats, err := svc.Statistics(ctx, collectorA)
	if err != nil {
		t.Fatalf("statistics: %v", err)
	}

	owned := statusOf(t, stats, collectible.StatusOwned)
	if owned.Total.String() != "0.60" {
		t.Errorf("0.10 + 0.20 + 0.30 = %q, want exactly \"0.60\". A float in the path gives "+
			"0.6000000000000001, or gives 0.60 here and is wrong by a cent elsewhere", owned.Total.String())
	}
}

// T026: collectibles with no price are excluded from totals and are not treated as free.
func TestUnpricedCollectiblesAreExcludedNotZeroed(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	for _, d := range []collectible.Draft{
		priced("u-1", "Has a price", "owned", "40.00", "", ""),
		priced("u-2", "No price at all", "owned", "", "", ""),
		priced("u-3", "Also no price", "owned", "", "", ""),
	} {
		if _, v, err := svc.Add(ctx, collectorA, d); err != nil || len(v) > 0 {
			t.Fatalf("add %q: %v %+v", d.Name, err, v)
		}
	}

	stats, err := svc.Statistics(ctx, collectorA)
	if err != nil {
		t.Fatalf("statistics: %v", err)
	}

	if stats.TotalCount != 3 {
		t.Errorf("totalCount = %d, want 3 — an unpriced collectible is still a collectible", stats.TotalCount)
	}
	if stats.PricedCount != 1 {
		t.Errorf("pricedCount = %d, want 1", stats.PricedCount)
	}
	owned := statusOf(t, stats, collectible.StatusOwned)
	if owned.Total.String() != "40.00" {
		t.Errorf("owned total = %q, want \"40.00\"; the two unpriced ones must contribute nothing "+
			"rather than contributing zero", owned.Total.String())
	}
}

// T027: an empty vault is a success carrying zeros.
func TestStatisticsOfAnEmptyVault(t *testing.T) {
	_, svc := freshStore(t)

	stats, err := svc.Statistics(context.Background(), collectorA)
	if err != nil {
		t.Fatalf("statistics of an empty vault: %v — a collector with nothing has a vault, and it "+
			"is empty", err)
	}
	if stats.TotalCount != 0 || stats.PricedCount != 0 {
		t.Errorf("empty vault = (%d, %d), want zeros", stats.TotalCount, stats.PricedCount)
	}
	if len(stats.ByStatus) != 4 {
		t.Errorf("byStatus has %d entries, want 4 even for an empty vault", len(stats.ByStatus))
	}
	for _, s := range stats.ByStatus {
		if s.Count != 0 || s.Total.String() != "0.00" {
			t.Errorf("%s = (%d, %q), want a zero entry", s.Status, s.Count, s.Total.String())
		}
	}
	if len(stats.BySeries) != 0 || len(stats.ByManufacturer) != 0 {
		t.Errorf("breakdowns over nothing should be empty, got %d and %d",
			len(stats.BySeries), len(stats.ByManufacturer))
	}
}

// T028: one collector's figures are untouched by another's vault (FR-002, SC-006).
func TestStatisticsAreIsolatedBetweenCollectors(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	if _, v, err := svc.Add(ctx, collectorA, priced("i-a", "A's piece", "owned", "100.00", "A Series", "A Maker")); err != nil || len(v) > 0 {
		t.Fatalf("add for A: %v %+v", err, v)
	}
	for i := 0; i < 5; i++ {
		d := priced(fmt.Sprintf("i-b-%d", i), fmt.Sprintf("B's piece %d", i), "owned", "999.00", "B Series", "B Maker")
		if _, v, err := svc.Add(ctx, collectorB, d); err != nil || len(v) > 0 {
			t.Fatalf("add for B: %v %+v", err, v)
		}
	}

	stats, err := svc.Statistics(ctx, collectorA)
	if err != nil {
		t.Fatalf("statistics: %v", err)
	}
	if stats.TotalCount != 1 {
		t.Errorf("A's totalCount = %d, want 1 — B's five are not A's", stats.TotalCount)
	}
	if got := statusOf(t, stats, collectible.StatusOwned); got.Total.String() != "100.00" {
		t.Errorf("A's owned total = %q, want \"100.00\"", got.Total.String())
	}
	if _, found := groupLabelled(stats.BySeries, "B Series"); found {
		t.Error("A's breakdown names B's series; every statement must carry collector_id")
	}
}

// T042 — FR-017, FR-017a, SC-003: every figure in one response describes the same vault.
//
// Three properties, which fail together if the aggregates ever drift apart: the per-status counts
// sum to the total, the per-status totals sum to the total paid, and both breakdowns account for
// every collectible too.
//
// The single-snapshot requirement (FR-017a) is structural rather than observable: the three
// aggregates run inside one read-only transaction, so there is no interleaving for a test to
// provoke. What is testable is the consequence — that the figures reconcile — and that is what this
// asserts. A future change that split the reads would not fail here deterministically; it would fail
// here *sometimes*, which is the reason the transaction exists.
func TestEveryAggregateAccountsForTheWholeVault(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	rows := []struct {
		status, price, series, manufacturer string
	}{
		{"owned", "10.00", "Alpha", "Maker One"},
		{"owned", "20.50", "Alpha", "Maker Two"},
		{"owned", "", "Beta", ""},
		{"preordered", "5.25", "Beta", "Maker One"},
		{"wishlist", "100.00", "", "Maker Two"},
		{"sold", "7.75", "Alpha", ""},
		{"sold", "", "", ""},
	}
	for i, r := range rows {
		d := priced(fmt.Sprintf("rec-%d", i), fmt.Sprintf("Piece %d", i), r.status, r.price, r.series, r.manufacturer)
		if _, v, err := svc.Add(ctx, collectorA, d); err != nil || len(v) > 0 {
			t.Fatalf("add %d: %v %+v", i, err, v)
		}
	}

	stats, err := svc.Statistics(ctx, collectorA)
	if err != nil {
		t.Fatalf("statistics: %v", err)
	}

	count, priced, total, err := collectible.Totals(stats.ByStatus)
	if err != nil {
		t.Fatalf("sum the status summaries: %v", err)
	}
	if count != stats.TotalCount {
		t.Errorf("the four status counts sum to %d but the vault reports %d", count, stats.TotalCount)
	}
	if priced != stats.PricedCount {
		t.Errorf("the four priced counts sum to %d but the vault reports %d", priced, stats.PricedCount)
	}
	// 10.00 + 20.50 + 5.25 + 100.00 + 7.75, stated rather than recomputed: a test that derives the
	// expected value the same way the code does agrees with the code even when both are wrong.
	if got, want := total.String(), "143.50"; got != want {
		t.Errorf("the status totals sum to %s, want %s", got, want)
	}

	for _, breakdown := range []struct {
		name   string
		groups []collectible.GroupSummary
	}{
		{"series", stats.BySeries},
		{"manufacturer", stats.ByManufacturer},
	} {
		var n, p int
		for _, g := range breakdown.groups {
			n += g.Count
			p += g.PricedCount
		}
		if n != stats.TotalCount {
			t.Errorf("the %s groups account for %d collectibles but the vault holds %d (FR-026)",
				breakdown.name, n, stats.TotalCount)
		}
		if p != stats.PricedCount {
			t.Errorf("the %s groups account for %d priced collectibles but the vault has %d",
				breakdown.name, p, stats.PricedCount)
		}
	}
}

// T043 — SC-007, FR-015: a status holding nothing is reported as zero, not omitted.
//
// The empty-vault case is covered elsewhere. This is the harder one: a vault with plenty in it where
// three statuses happen to be empty. The database returns no row for them, and a client that had to
// infer "absent means zero" would be deriving a figure it was never told.
func TestAStatusHoldingNothingIsReportedAsZero(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		d := priced(fmt.Sprintf("z-%d", i), fmt.Sprintf("Owned %d", i), "owned", "9.99", "Alpha", "Maker One")
		if _, v, err := svc.Add(ctx, collectorA, d); err != nil || len(v) > 0 {
			t.Fatalf("add %d: %v %+v", i, err, v)
		}
	}

	stats, err := svc.Statistics(ctx, collectorA)
	if err != nil {
		t.Fatalf("statistics: %v", err)
	}
	if len(stats.ByStatus) != 4 {
		t.Fatalf("got %d status summaries, want 4 — always four, whatever the vault holds", len(stats.ByStatus))
	}

	for _, status := range []collectible.CollectionStatus{"preordered", "wishlist", "sold"} {
		row := statusOf(t, stats, status)
		if row.Count != 0 || row.PricedCount != 0 || row.Total.String() != "0.00" {
			t.Errorf("%s holds nothing but reports count=%d priced=%d total=%s",
				status, row.Count, row.PricedCount, row.Total)
		}
	}
	if owned := statusOf(t, stats, "owned"); owned.Count != 3 || owned.Total.String() != "29.97" {
		t.Errorf("owned reports count=%d total=%s, want 3 and 29.97", owned.Count, owned.Total)
	}
}

// T044: a collectible that changes status moves between subtotals, and takes its money with it.
//
// This is the property a cached or precomputed total would break. Nothing here is stored; the
// figures are read from the collectibles themselves each time, so an edit is reflected immediately
// (FR-039).
func TestChangingStatusMovesACollectibleBetweenSubtotals(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	added, v, err := svc.Add(ctx, collectorA, priced("mv-1", "Moves", "preordered", "250.00", "Alpha", "Maker One"))
	if err != nil || len(v) > 0 {
		t.Fatalf("add: %v %+v", err, v)
	}

	before, err := svc.Statistics(ctx, collectorA)
	if err != nil {
		t.Fatalf("statistics before: %v", err)
	}
	if got := statusOf(t, before, "preordered"); got.Count != 1 || got.Total.String() != "250.00" {
		t.Fatalf("preordered before the edit: count=%d total=%s, want 1 and 250.00", got.Count, got.Total)
	}
	if got := statusOf(t, before, "owned"); got.Count != 0 || got.Total.String() != "0.00" {
		t.Fatalf("owned before the edit: count=%d total=%s, want 0 and 0.00", got.Count, got.Total)
	}

	e := edit(added.Row.Collectible, added.Row.Collectible.Version)
	e.Status = "owned"
	if _, v, err := svc.Edit(ctx, collectorA, added.Row.Collectible.ID, e); err != nil || len(v) > 0 {
		t.Fatalf("edit: %v %+v", err, v)
	}

	after, err := svc.Statistics(ctx, collectorA)
	if err != nil {
		t.Fatalf("statistics after: %v", err)
	}
	if got := statusOf(t, after, "preordered"); got.Count != 0 || got.Total.String() != "0.00" {
		t.Errorf("preordered after the edit: count=%d total=%s, want 0 and 0.00 — the piece left",
			got.Count, got.Total)
	}
	if got := statusOf(t, after, "owned"); got.Count != 1 || got.Total.String() != "250.00" {
		t.Errorf("owned after the edit: count=%d total=%s, want 1 and 250.00 — the money moved with it",
			got.Count, got.Total)
	}
	if after.TotalCount != 1 {
		t.Errorf("the vault holds %d pieces after a status change, want 1", after.TotalCount)
	}
}
