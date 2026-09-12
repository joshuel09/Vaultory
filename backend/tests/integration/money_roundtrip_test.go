//go:build integration

package integration

import (
	"context"
	"testing"
)

// FR-016, SC-009: an amount comes back exactly as it went in.
//
// numeric(12,2) plus a decimal string across every boundary. If anything in the path ever converts
// through a float, one of these will drift by a cent.
func TestPurchasePriceRoundTripsExactly(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	amounts := []string{
		"0.00", // a gift: a recorded amount, not an absence
		"0.01", // smallest
		"0.99",
		"1.00",
		"249.99",
		"1250.00",
		"99999.95",
		"1234567.89",
		"9999999999.99", // the largest numeric(12,2) holds
	}

	for i, amount := range amounts {
		d := draft("money-"+amount, "Priced Collectible")
		value := amount
		d.PurchasePrice = &value
		if _, v, err := svc.Add(ctx, collectorA, d); err != nil || len(v) > 0 {
			t.Fatalf("add %s: %v %+v", amount, err, v)
		}
		_ = i
	}

	page, err := svc.List(ctx, collectorA, nil, "", 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Rows) != len(amounts) {
		t.Fatalf("stored %d collectibles, want %d", len(page.Rows), len(amounts))
	}

	got := map[string]bool{}
	for _, r := range page.Rows {
		if r.Collectible.PurchasePrice == nil {
			t.Fatal("a recorded price came back absent")
		}
		got[r.Collectible.PurchasePrice.String()] = true
	}
	for _, amount := range amounts {
		if !got[amount] {
			t.Errorf("%s did not survive the round trip exactly; stored values were %v", amount, keys(got))
		}
	}
}

// FR-007: a recorded zero and no amount at all are different things, all the way to storage.
func TestAbsentPriceIsNotZero(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	zero := "0.00"
	withZero := draft("zero", "Gift")
	withZero.PurchasePrice = &zero
	if _, _, err := svc.Add(ctx, collectorA, withZero); err != nil {
		t.Fatalf("add with zero: %v", err)
	}
	if _, _, err := svc.Add(ctx, collectorA, draft("none", "No price recorded")); err != nil {
		t.Fatalf("add without price: %v", err)
	}

	page, err := svc.List(ctx, collectorA, nil, "", 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var recordedZero, absent int
	for _, r := range page.Rows {
		switch {
		case r.Collectible.PurchasePrice == nil:
			absent++
		case r.Collectible.PurchasePrice.IsZero():
			recordedZero++
		}
	}
	if recordedZero != 1 || absent != 1 {
		t.Errorf("recorded-zero=%d absent=%d, want 1 and 1 — these must stay distinguishable (FR-007)",
			recordedZero, absent)
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
