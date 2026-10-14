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
