//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
)

// FR-034, FR-035: pages tile the collection newest-first, with nothing repeated or skipped.
func TestKeysetPaginationTilesTheCollection(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	const total = 57
	for i := 0; i < total; i++ {
		if _, v, err := svc.Add(ctx, collectorA, draft(fmt.Sprintf("page-%d", i), fmt.Sprintf("Collectible %02d", i))); err != nil || len(v) > 0 {
			t.Fatalf("seed %d: %v %+v", i, err, v)
		}
	}

	seen := map[string]int{}
	var order []string
	cursor := ""
	pages := 0
	for {
		page, err := svc.List(ctx, collectorA, nil, cursor, 10)
		if err != nil {
			t.Fatalf("page %d: %v", pages, err)
		}
		pages++
		if page.TotalUnfiltered != total {
			t.Errorf("page %d: totalUnfiltered = %d, want %d on every page", pages, page.TotalUnfiltered, total)
		}
		for _, r := range page.Rows {
			id := r.Collectible.ID.String()
			seen[id]++
			order = append(order, r.Collectible.Name)
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
		if pages > 20 {
			t.Fatal("pagination did not terminate")
		}
	}

	if len(seen) != total {
		t.Errorf("saw %d distinct collectibles across %d pages, want %d", len(seen), pages, total)
	}
	for id, n := range seen {
		if n != 1 {
			t.Errorf("collectible %s appeared %d times across pages, want once", id, n)
		}
	}

	// Newest first: the last one added leads.
	if len(order) > 0 && order[0] != fmt.Sprintf("Collectible %02d", total-1) {
		t.Errorf("first entry is %q, want the most recently added (FR-034)", order[0])
	}
}

// Entries created in the same instant must still be ordered totally, or a page boundary landing
// between them would skip or repeat one. The id tiebreaker is what provides that.
func TestPaginationIsStableAcrossIdenticalTimestamps(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	const total = 20
	for i := 0; i < total; i++ {
		if _, _, err := svc.Add(ctx, collectorA, draft(fmt.Sprintf("same-%d", i), "Simultaneous")); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
	// Collapse every timestamp so created_at alone cannot order them.
	if _, err := pool.Exec(ctx, `UPDATE collectibles SET created_at = now()`); err != nil {
		t.Fatalf("collapse timestamps: %v", err)
	}

	seen := map[string]int{}
	cursor := ""
	for {
		page, err := svc.List(ctx, collectorA, nil, cursor, 3)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		for _, r := range page.Rows {
			seen[r.Collectible.ID.String()]++
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}

	if len(seen) != total {
		t.Errorf("saw %d of %d collectibles when every timestamp is identical", len(seen), total)
	}
	for id, n := range seen {
		if n != 1 {
			t.Errorf("collectible %s appeared %d times, want once", id, n)
		}
	}
}

// A cursor the client invented or corrupted is a client error, not a server failure.
func TestMalformedCursorIsRejected(t *testing.T) {
	_, svc := freshStore(t)
	if _, err := svc.List(context.Background(), collectorA, nil, "not-a-cursor", 10); err == nil {
		t.Error("a malformed cursor should be rejected")
	}
}
