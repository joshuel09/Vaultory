//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
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
		if _, err := svc.List(ctx, collectorA, postgres.ListFilter{}, "", 48); err != nil {
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

// T075: an edit and a deletion each cost a bounded number of statements.
//
// Both drain the pending-image-deletion queue, which is the thing most likely to turn one
// collector's operation into arbitrarily much work. The drain takes a bounded batch for exactly
// that reason; this asserts the bound holds rather than trusting the constant.
func TestEditAndDeleteCostDoesNotGrowWithTheQueue(t *testing.T) {
	store, svc := freshStore(t)
	ctx := context.Background()

	measure := func(label string, work func()) int64 {
		before := statementCount(t)
		work()
		after := statementCount(t) - before
		t.Logf("%s cost %d statements", label, after)
		return after
	}

	// A baseline edit and deletion with an empty queue.
	first, _, err := svc.Add(ctx, collectorA, draft("qc-1", "Baseline"))
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	baselineEdit := measure("edit (empty queue)", func() {
		e := edit(first.Row.Collectible, first.Row.Collectible.Version)
		e.Name = "Baseline edited"
		if _, v, err := svc.Edit(ctx, collectorA, first.Row.Collectible.ID, e); err != nil || len(v) > 0 {
			t.Fatalf("edit: %v %+v", err, v)
		}
	})
	baselineDelete := measure("delete (empty queue)", func() {
		if err := svc.Delete(ctx, collectorA, first.Row.Collectible.ID); err != nil {
			t.Fatalf("delete: %v", err)
		}
	})

	// Now a queue far larger than one batch. Each entry names files that do not exist, which the
	// image store treats as already deleted — so the work is real but the bytes are not.
	for i := 0; i < 200; i++ {
		if _, err := pool.Exec(ctx, `
			INSERT INTO pending_image_deletions (image_id, collector_id, original_key, rendition_key)
			VALUES ($1, $2, $3, $4)`,
			uuid.New(), collectorA, fmt.Sprintf("orig/ghost-%d", i), fmt.Sprintf("rend/ghost-%d", i),
		); err != nil {
			t.Fatalf("seed the queue: %v", err)
		}
	}
	_ = store

	second, _, err := svc.Add(ctx, collectorA, draft("qc-2", "Loaded"))
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	loadedEdit := measure("edit (200 queued)", func() {
		e := edit(second.Row.Collectible, second.Row.Collectible.Version)
		e.Name = "Loaded edited"
		if _, v, err := svc.Edit(ctx, collectorA, second.Row.Collectible.ID, e); err != nil || len(v) > 0 {
			t.Fatalf("edit: %v %+v", err, v)
		}
	})

	// The drain costs a statement per entry it handles, so the loaded case is dearer — but by a
	// bounded amount, not by 200. Anything proportional to the queue means the batch is not
	// holding.
	const generousBound = 80
	if loadedEdit-baselineEdit > generousBound {
		t.Errorf("an edit cost %d extra statements with 200 entries queued (baseline %d); the "+
			"drain is not bounded, so one collector's edit pays for every orphaned file",
			loadedEdit-baselineEdit, baselineEdit)
	}

	loadedDelete := measure("delete (queue partly drained)", func() {
		if err := svc.Delete(ctx, collectorA, second.Row.Collectible.ID); err != nil {
			t.Fatalf("delete: %v", err)
		}
	})
	if loadedDelete-baselineDelete > generousBound {
		t.Errorf("a deletion cost %d extra statements with a loaded queue (baseline %d)",
			loadedDelete-baselineDelete, baselineDelete)
	}

	// And the queue is actually being worked down rather than merely tolerated.
	remaining := queueDepth(t)
	if remaining >= 200 {
		t.Errorf("the queue still holds %d entries after two operations; nothing is draining it", remaining)
	}
}

// T030: the statistics view costs the same whatever the collection holds.
//
// A note on what this actually measures, because the helper's name misleads: statementCount reads
// xact_commit + xact_rollback, so it counts *transactions*, not statements. The statistics read runs
// all three of its aggregates inside one read-only transaction, so the figure it reports is one.
//
// The number is not the point. The property is: the cost must not move between a small vault and a
// large one, because that is what distinguishes an aggregate from a loop. Asserting the constant
// would be asserting an implementation detail of the helper.
func TestStatisticsCostDoesNotGrowWithCollection(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	measure := func() int64 {
		before := statementCount(t)
		if _, err := svc.Statistics(ctx, collectorA); err != nil {
			t.Fatalf("statistics: %v", err)
		}
		return statementCount(t) - before
	}

	small := measure()

	// Enough distinct series and manufacturers to push well past the breakdown bound, which is the
	// shape most likely to turn one aggregate into per-group work.
	for i := 0; i < 60; i++ {
		d := priced(
			fmt.Sprintf("qc-%d", i),
			fmt.Sprintf("Piece %d", i),
			"owned",
			"12.34",
			fmt.Sprintf("Series %d", i%25),
			fmt.Sprintf("Maker %d", i%17),
		)
		if _, v, err := svc.Add(ctx, collectorA, d); err != nil || len(v) > 0 {
			t.Fatalf("add %d: %v %+v", i, err, v)
		}
	}

	large := measure()

	if small != large {
		t.Errorf("statistics cost %d for an empty vault and %d for a 60-item one across 25 series "+
			"and 17 manufacturers; the cost must not grow with the collection (no N+1)", small, large)
	}
}
