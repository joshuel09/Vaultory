//go:build integration

package integration

import (
	"context"
	"testing"
)

// FR-023, FR-024: two deliberate copies of the same collectible stay independent. They are never
// merged, de-duplicated, or shown as a quantity — a collector who owns two of a figure owns two
// entries, each with its own price, status, and notes.
func TestDeliberateDuplicatesStayIndependent(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	// Identical in every respect except the submission key, which is what marks each as a
	// deliberate add rather than a retry (FR-047).
	first, v, err := svc.Add(ctx, collectorA, draft("dup-1", "Kaiju Sentinel"))
	if err != nil || len(v) > 0 {
		t.Fatalf("first add: %v %+v", err, v)
	}
	second, v, err := svc.Add(ctx, collectorA, draft("dup-2", "Kaiju Sentinel"))
	if err != nil || len(v) > 0 {
		t.Fatalf("second add: %v %+v", err, v)
	}

	if first.Row.Collectible.ID == second.Row.Collectible.ID {
		t.Fatal("two deliberate adds produced one collectible; FR-023 forbids de-duplication")
	}
	if first.Existing || second.Existing {
		t.Error("neither add should have been treated as a replay")
	}

	// A third copy, sold, at a different price — each entry carries its own values (FR-024).
	price := "1450.00"
	third := draft("dup-3", "Kaiju Sentinel")
	third.Status = "sold"
	third.PurchasePrice = &price
	if _, v, err := svc.Add(ctx, collectorA, third); err != nil || len(v) > 0 {
		t.Fatalf("third add: %v %+v", err, v)
	}

	page, err := svc.List(ctx, collectorA, nil, "", 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Rows) != 3 {
		t.Fatalf("collection has %d entries, want 3 independent ones", len(page.Rows))
	}
	if page.TotalUnfiltered != 3 {
		t.Errorf("totalUnfiltered = %d, want 3", page.TotalUnfiltered)
	}

	var sold int
	for _, r := range page.Rows {
		if r.Collectible.Status == "sold" {
			sold++
			if r.Collectible.PurchasePrice == nil || r.Collectible.PurchasePrice.String() != "1450.00" {
				t.Error("the sold copy lost its own purchase price")
			}
		}
	}
	if sold != 1 {
		t.Errorf("%d sold entries, want exactly 1 — statuses must not be shared between copies", sold)
	}
}
