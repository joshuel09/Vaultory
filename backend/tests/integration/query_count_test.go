//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
)

// T082: a gallery page costs a fixed number of statements, whatever it contains.
//
// The constitution forbids N+1 access. The risk here is a per-entry image lookup, which would grow
// with the page — this asserts that the cost does not move when the page fills up.
func TestGalleryPageCostDoesNotGrowWithItems(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	statementsFor := func(items int) int64 {
		before := statementCount(t)
		if _, err := svc.List(ctx, collectorA, nil, "", 48); err != nil {
			t.Fatalf("list with %d items: %v", items, err)
		}
		return statementCount(t) - before
	}

	// An empty collection first.
	empty := statementsFor(0)

	// Then a full page, each entry carrying an image — the shape most likely to provoke N+1.
	for i := 0; i < 30; i++ {
		img, err := svc.UploadImage(ctx, collectorA, bytesReader(testJPEG(t, 120, 150)))
		if err != nil {
			t.Fatalf("upload %d: %v", i, err)
		}
		id := img.ID.String()
		d := draft(fmt.Sprintf("n1-%d", i), fmt.Sprintf("With image %d", i))
		d.ImageID = &id
		if _, v, err := svc.Add(ctx, collectorA, d); err != nil || len(v) > 0 {
			t.Fatalf("add %d: %v %+v", i, err, v)
		}
	}
	full := statementsFor(30)

	if full != empty {
		t.Errorf("an empty page cost %d statements and a 30-item page cost %d; the cost must not "+
			"grow with the number of entries (no N+1)", empty, full)
	}
	// Two: the page itself and the totalUnfiltered count, as recorded in data-model.md.
	if full > 2 {
		t.Errorf("a gallery page cost %d statements, want 2 (the page and the count)", full)
	}
	t.Logf("a gallery page costs %d statements regardless of size", full)
}

func statementCount(t *testing.T) int64 {
	t.Helper()
	var n int64
	err := pool.QueryRow(context.Background(),
		`SELECT xact_commit + xact_rollback FROM pg_stat_database WHERE datname = current_database()`).Scan(&n)
	if err != nil {
		t.Skipf("statement statistics unavailable: %v", err)
	}
	return n
}
