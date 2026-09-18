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
	// Only the pool is needed: this test seeds directly and inspects a query plan.
	_, _ = freshStore(t)
	ctx := context.Background()

	// Enough rows that the planner has a real decision to make.
	//
	// The previous version of this test seeded 40 and then asserted the index was used, while its
	// own comment admitted a scan was reasonable at that size. It was: on 40 rows PostgreSQL reads
	// the heap and sorts, because that is genuinely cheaper, and the test failed for being wrong
	// rather than for the index being wrong. An index is only worth asserting at a size where not
	// using it would cost something.
	//
	// Inserted directly rather than through the service: 3000 round trips would dominate the
	// suite's runtime, and what is under test is the query plan, not the write path.
	if _, err := pool.Exec(ctx, `
		INSERT INTO collectibles (collector_id, name, collection_status, created_at)
		SELECT $1, 'Indexed ' || g, CASE g % 4
		         WHEN 0 THEN 'owned' WHEN 1 THEN 'preordered'
		         WHEN 2 THEN 'wishlist' ELSE 'sold' END,
		       now() - make_interval(secs => g)
		FROM generate_series(1, 3000) AS g`, collectorA); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// Statistics, or the planner is choosing from defaults rather than from this table.
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
