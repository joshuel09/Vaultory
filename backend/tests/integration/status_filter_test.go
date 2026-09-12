//go:build integration

package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
)

// FR-037, FR-040: each status returns only its own entries, and totalUnfiltered stays independent
// of the filter — which is what lets the frontend tell an empty vault from a filter that matched
// nothing.
func TestStatusFilter(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	counts := map[string]int{"owned": 4, "preordered": 3, "wishlist": 2, "sold": 0}
	total := 0
	for status, n := range counts {
		for i := 0; i < n; i++ {
			d := draft(fmt.Sprintf("%s-%d", status, i), fmt.Sprintf("%s %d", status, i))
			d.Status = status
			if _, v, err := svc.Add(ctx, collectorA, d); err != nil || len(v) > 0 {
				t.Fatalf("seed %s: %v %+v", status, err, v)
			}
			total++
		}
	}

	for status, want := range counts {
		parsed, err := collectible.ParseCollectionStatus(status)
		if err != nil {
			t.Fatalf("parse %s: %v", status, err)
		}
		page, err := svc.List(ctx, collectorA, &parsed, "", 50)
		if err != nil {
			t.Fatalf("list %s: %v", status, err)
		}
		if len(page.Rows) != want {
			t.Errorf("status %s returned %d entries, want %d", status, len(page.Rows), want)
		}
		for _, r := range page.Rows {
			if string(r.Collectible.Status) != status {
				t.Errorf("filtering by %s returned a %s entry", status, r.Collectible.Status)
			}
		}
		// FR-040: the count ignores the filter, so zero results with a non-zero total is a
		// no-results state rather than an empty vault.
		if page.TotalUnfiltered != total {
			t.Errorf("status %s: totalUnfiltered = %d, want %d regardless of the filter",
				status, page.TotalUnfiltered, total)
		}
	}

	// Sold matched nothing while the vault holds nine collectibles: the exact situation the
	// no-results state exists for.
	sold, _ := collectible.ParseCollectionStatus("sold")
	page, err := svc.List(ctx, collectorA, &sold, "", 50)
	if err != nil {
		t.Fatalf("list sold: %v", err)
	}
	if len(page.Rows) != 0 || page.TotalUnfiltered == 0 {
		t.Errorf("expected an empty page with a non-zero total, got %d rows and total %d",
			len(page.Rows), page.TotalUnfiltered)
	}
}

// The filtered gallery query must use the status index, not a sequential scan. This is the
// constitution's requirement that large collection views have an efficient strategy.
func TestFilteredGalleryQueryUsesTheStatusIndex(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	for i := 0; i < 40; i++ {
		if _, _, err := svc.Add(ctx, collectorA, draft(fmt.Sprintf("idx-%d", i), "Indexed")); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	// Planner statistics, or it may reasonably choose a scan on a tiny table.
	if _, err := pool.Exec(ctx, `ANALYZE collectibles`); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	rows, err := pool.Query(ctx, `
		EXPLAIN SELECT c.id FROM collectibles c
		WHERE c.collector_id = $1 AND c.collection_status = $2
		ORDER BY c.created_at DESC, c.id DESC LIMIT 25`,
		collectorA, "owned")
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	defer rows.Close()

	var plan strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("scan plan: %v", err)
		}
		plan.WriteString(line)
		plan.WriteString("\n")
	}
	if !strings.Contains(plan.String(), "collectibles_status_gallery_idx") {
		t.Logf("query plan:\n%s", plan.String())
		t.Error("the filtered gallery query did not use collectibles_status_gallery_idx")
	}
}
