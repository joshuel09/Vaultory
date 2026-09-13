//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// T081, SC-004: a 500-collectible collection becomes browsable within 2 seconds.
//
// "Browsable" means the first page arrives, not the whole collection — the collection is paged
// precisely so it never has to arrive at once (FR-035).
func TestLargeCollectionIsBrowsableQuickly(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	const total = 500
	for i := 0; i < total; i++ {
		if _, _, err := svc.Add(ctx, collectorA, draft(fmt.Sprintf("perf-%d", i), fmt.Sprintf("Collectible %03d", i))); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
	if _, err := pool.Exec(ctx, `ANALYZE collectibles`); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	start := time.Now()
	page, err := svc.List(ctx, collectorA, nil, "", 24)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Rows) != 24 {
		t.Fatalf("first page has %d entries, want 24", len(page.Rows))
	}
	if page.TotalUnfiltered != total {
		t.Errorf("total = %d, want %d", page.TotalUnfiltered, total)
	}
	// A generous ceiling: the budget in SC-004 is for the whole round trip including rendering,
	// and this measures only the data. If this alone approaches the budget, something is wrong.
	if elapsed > 500*time.Millisecond {
		t.Errorf("the first page of a %d-collectible collection took %v; SC-004 budgets 2s for the "+
			"entire view, so the query alone must be far below that", total, elapsed)
	}
	t.Logf("first page of %d collectibles in %v", total, elapsed)

	// Paging deep into the collection must cost no more than paging into the start of it — that is
	// the property keyset pagination buys over offset.
	cursor := page.NextCursor
	for i := 0; i < 15 && cursor != ""; i++ {
		p, err := svc.List(ctx, collectorA, nil, cursor, 24)
		if err != nil {
			t.Fatalf("page %d: %v", i, err)
		}
		cursor = p.NextCursor
	}
	if cursor == "" {
		t.Skip("collection exhausted before a deep page could be measured")
	}
	start = time.Now()
	if _, err := svc.List(ctx, collectorA, nil, cursor, 24); err != nil {
		t.Fatalf("deep page: %v", err)
	}
	deep := time.Since(start)
	t.Logf("a deep page took %v (first page: %v)", deep, elapsed)
	if deep > 4*elapsed+100*time.Millisecond {
		t.Errorf("a deep page took %v against %v for the first; keyset pagination should keep page "+
			"cost roughly constant", deep, elapsed)
	}
}
