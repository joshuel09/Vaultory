package unit

import (
	"testing"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
)

func money(t *testing.T, raw string) collectible.Money {
	t.Helper()
	m, err := collectible.ParseMoney(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return m
}

// T016: a vault using two statuses still yields four summaries.
//
// A status holding nothing produces no database row. Omitting it would make "holds nothing" and
// "was not returned" indistinguishable to anything downstream (FR-015).
func TestEveryStatusIsPresentEvenWhenEmpty(t *testing.T) {
	got := collectible.FillMissingStatuses([]collectible.StatusSummary{
		{Status: collectible.StatusSold, Count: 2, PricedCount: 1, Total: money(t, "30.00")},
		{Status: collectible.StatusOwned, Count: 5, PricedCount: 5, Total: money(t, "100.00")},
	})

	if len(got) != 4 {
		t.Fatalf("got %d summaries, want exactly 4", len(got))
	}

	want := collectible.AllStatuses()
	for i, s := range got {
		if s.Status != want[i] {
			t.Errorf("summary %d is %q, want %q — the order must be stable so a client can rely on it",
				i, s.Status, want[i])
		}
	}

	byStatus := map[collectible.CollectionStatus]collectible.StatusSummary{}
	for _, s := range got {
		byStatus[s.Status] = s
	}

	if s := byStatus[collectible.StatusOwned]; s.Count != 5 || s.Total.String() != "100.00" {
		t.Errorf("owned = %+v, want the values that were supplied", s)
	}
	if s := byStatus[collectible.StatusWishlist]; s.Count != 0 || s.PricedCount != 0 || s.Total.String() != "0.00" {
		t.Errorf("wishlist = %+v, want a zero entry rather than an absent one", s)
	}
}

// T017: ordering puts a large remainder below a small named group.
//
// This is the case that ordering by count alone gets wrong, and it was found by running the query
// against real data rather than by reasoning about it.
func TestRemainderAndUnspecifiedSortBelowEveryNamedGroup(t *testing.T) {
	groups := []collectible.GroupSummary{
		{Label: "Everything else", Kind: collectible.GroupRemainder, Count: 40},
		{Label: "Not specified", Kind: collectible.GroupUnspecified, Count: 25},
		{Label: "Alter", Kind: collectible.GroupNamed, Count: 3},
		{Label: "Good Smile Company", Kind: collectible.GroupNamed, Count: 9},
	}
	collectible.SortGroups(groups)

	wantLabels := []string{"Good Smile Company", "Alter", "Not specified", "Everything else"}
	for i, want := range wantLabels {
		if groups[i].Label != want {
			t.Errorf("position %d is %q, want %q; a remainder of 40 must still sit below a named "+
				"group of 3, or the breakdown reads as though the collection is mostly nothing in "+
				"particular", i, groups[i].Label, want)
		}
	}
}

// T017: named groups that tie on count come back in the same order every time (FR-023).
func TestTiedGroupsOrderDeterministically(t *testing.T) {
	first := []collectible.GroupSummary{
		{Label: "Kotobukiya", Kind: collectible.GroupNamed, Count: 4},
		{Label: "Alter", Kind: collectible.GroupNamed, Count: 4},
		{Label: "Max Factory", Kind: collectible.GroupNamed, Count: 4},
	}
	second := []collectible.GroupSummary{
		{Label: "Max Factory", Kind: collectible.GroupNamed, Count: 4},
		{Label: "Kotobukiya", Kind: collectible.GroupNamed, Count: 4},
		{Label: "Alter", Kind: collectible.GroupNamed, Count: 4},
	}
	collectible.SortGroups(first)
	collectible.SortGroups(second)

	for i := range first {
		if first[i].Label != second[i].Label {
			t.Fatalf("two differently-ordered inputs sorted differently at %d: %q vs %q; the view "+
				"would reshuffle between loads", i, first[i].Label, second[i].Label)
		}
	}
	if first[0].Label != "Alter" {
		t.Errorf("ties resolve to %q, want alphabetical", first[0].Label)
	}
}

// T018: pricedCount distinguishes "nothing is priced" from "things that cost nothing".
//
// Both report a total of "0.00". Without pricedCount a collector reading a zero cannot tell which
// they are looking at, and FR-010 exists for exactly that reason.
func TestTotalsDistinguishUnpricedFromFree(t *testing.T) {
	nothingPriced := collectible.FillMissingStatuses([]collectible.StatusSummary{
		{Status: collectible.StatusOwned, Count: 3, PricedCount: 0, Total: money(t, "0.00")},
	})
	count, priced, total, err := collectible.Totals(nothingPriced)
	if err != nil {
		t.Fatalf("totals: %v", err)
	}
	if count != 3 || priced != 0 || total.String() != "0.00" {
		t.Errorf("nothing priced = (%d, %d, %q), want (3, 0, \"0.00\")", count, priced, total.String())
	}

	recordedAsFree := collectible.FillMissingStatuses([]collectible.StatusSummary{
		{Status: collectible.StatusOwned, Count: 3, PricedCount: 3, Total: money(t, "0.00")},
	})
	count, priced, total, err = collectible.Totals(recordedAsFree)
	if err != nil {
		t.Fatalf("totals: %v", err)
	}
	if count != 3 || priced != 3 || total.String() != "0.00" {
		t.Errorf("recorded as free = (%d, %d, %q), want (3, 3, \"0.00\")", count, priced, total.String())
	}
}

// T018: the whole-vault figures are the sum of the four status rows, exactly.
func TestTotalsSumTheStatusRowsExactly(t *testing.T) {
	byStatus := collectible.FillMissingStatuses([]collectible.StatusSummary{
		{Status: collectible.StatusOwned, Count: 2, PricedCount: 2, Total: money(t, "0.10")},
		{Status: collectible.StatusPreordered, Count: 1, PricedCount: 1, Total: money(t, "0.20")},
		{Status: collectible.StatusSold, Count: 4, PricedCount: 3, Total: money(t, "0.30")},
	})

	count, priced, total, err := collectible.Totals(byStatus)
	if err != nil {
		t.Fatalf("totals: %v", err)
	}
	if count != 7 {
		t.Errorf("count = %d, want 7", count)
	}
	if priced != 6 {
		t.Errorf("pricedCount = %d, want 6", priced)
	}
	// Written as a literal, not computed, so this cannot agree with a bug by reproducing it.
	if total.String() != "0.60" {
		t.Errorf("total = %q, want \"0.60\"", total.String())
	}
}
