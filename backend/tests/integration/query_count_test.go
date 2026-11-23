//go:build integration

package integration

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joshuel09/vaultory/backend/internal/collection"
	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
)

// T082: a gallery page costs a fixed number of statements, whatever it contains.
//
// The constitution forbids N+1 access. The risk here is a per-entry image lookup, which would grow
// with the page — this asserts that the cost does not move when the page fills up.
func TestGalleryPageCostDoesNotGrowWithItems(t *testing.T) {
	svc := freshTracedService(t)
	ctx := context.Background()

	statementsFor := func(items int) int64 {
		return countQueries(ctx, func(ctx context.Context) {
			if _, err := svc.List(ctx, collectorA, postgres.ListFilter{}, "", 48); err != nil {
				t.Fatalf("list with %d items: %v", items, err)
			}
		})
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
	if full != 2 {
		t.Errorf("a gallery page cost %d statements, want 2 (the page and the count)", full)
	}
	t.Logf("a gallery page costs %d statements regardless of size", full)
}

// queryCounterKey scopes a count to one service call, including derived contexts and
// any connection the pool selects. Fixtures and unrelated queries have no counter.
type queryCounterKey struct{}

type queryCountTracer struct{}

var _ pgx.QueryTracer = queryCountTracer{}

// TraceQueryStart counts data queries only when their context carries a measurement.
func (queryCountTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	counter, ok := ctx.Value(queryCounterKey{}).(*atomic.Int64)
	if !ok {
		return ctx
	}
	// A read transaction has BEGIN/ROLLBACK overhead, but the query budget concerns
	// data statements. Count every query within it rather than the transaction itself.
	words := strings.Fields(strings.ToLower(data.SQL))
	if len(words) > 0 {
		switch words[0] {
		case "begin", "commit", "rollback", "savepoint", "release":
			return ctx
		}
	}
	counter.Add(1)
	return ctx
}

// TraceQueryEnd needs no bookkeeping because query starts are counted synchronously.
func (queryCountTracer) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// countQueries measures one operation without including fixture work or other contexts.
func countQueries(ctx context.Context, work func(context.Context)) int64 {
	counter := &atomic.Int64{}
	work(context.WithValue(ctx, queryCounterKey{}, counter))
	return counter.Load()
}

// tracedPool copies the test database configuration and instruments every new connection.
func tracedPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	cfg := pool.Config()
	cfg.ConnConfig.Tracer = queryCountTracer{}
	db, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("create traced pool: %v", err)
	}
	t.Cleanup(db.Close)
	return db
}

// freshTracedService uses the standard fixtures with an operation-traced repository.
func freshTracedService(t *testing.T) *collection.Service {
	t.Helper()
	freshStore(t) // Reset and seed the same database as every other integration test.
	_, svc := serviceForPool(t, tracedPool(t))
	return svc
}

// The counter must see each query inside one transaction, across connections, and
// through derived contexts. Otherwise an N+1 loop can be invisible to the budget.
func TestQueryCounterMeasuresOnlyScopedQueries(t *testing.T) {
	db := tracedPool(t)
	ctx := context.Background()
	first, err := db.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Release()
	second, err := db.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Release()
	if first.Conn().PgConn().PID() == second.Conn().PgConn().PID() {
		t.Fatal("expected two distinct database connections")
	}

	n := countQueries(ctx, func(measured context.Context) {
		derived, cancel := context.WithCancel(measured)
		defer cancel()
		if _, err := first.Exec(derived, "SELECT 1"); err != nil {
			t.Fatal(err)
		}
		var value int
		if err := first.QueryRow(derived, "SELECT 2").Scan(&value); err != nil {
			t.Fatal(err)
		}
		rows, err := first.Query(derived, "SELECT 3")
		if err != nil {
			t.Fatal(err)
		}
		rows.Close()

		// This executes during the measurement but has no measurement context.
		if _, err := second.Exec(ctx, "SELECT 99"); err != nil {
			t.Fatal(err)
		}
		tx, err := second.BeginTx(derived, pgx.TxOptions{AccessMode: pgx.ReadOnly})
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(derived) }()
		for i := 0; i < 2; i++ {
			if err := tx.QueryRow(derived, "SELECT 4").Scan(&value); err != nil {
				t.Fatal(err)
			}
		}
		if err := tx.Commit(derived); err != nil {
			t.Fatal(err)
		}
	})
	if n != 5 {
		t.Fatalf("counted %d queries, want exactly 5 (excluding transaction control and unscoped queries)", n)
	}
	if n := countQueries(ctx, func(context.Context) {}); n != 0 {
		t.Fatalf("empty operation counted %d queries; measurement leaked between calls", n)
	}
}

// T075: an edit and a deletion each cost a bounded number of statements.
//
// Both drain the pending-image-deletion queue, which is the thing most likely to turn one
// collector's operation into arbitrarily much work. The drain takes a bounded batch for exactly
// that reason; this asserts the bound holds rather than trusting the constant.
func TestEditAndDeleteCostDoesNotGrowWithTheQueue(t *testing.T) {
	svc := freshTracedService(t)
	ctx := context.Background()

	measure := func(label string, work func(context.Context)) int64 {
		after := countQueries(ctx, work)
		t.Logf("%s cost %d statements", label, after)
		return after
	}

	// A baseline edit and deletion with an empty queue.
	first, _, err := svc.Add(ctx, collectorA, draft("qc-1", "Baseline"))
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	baselineEdit := measure("edit (empty queue)", func(ctx context.Context) {
		e := edit(first.Row.Collectible, first.Row.Collectible.Version)
		e.Name = "Baseline edited"
		if _, v, err := svc.Edit(ctx, collectorA, first.Row.Collectible.ID, e); err != nil || len(v) > 0 {
			t.Fatalf("edit: %v %+v", err, v)
		}
	})
	baselineDelete := measure("delete (empty queue)", func(ctx context.Context) {
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

	second, _, err := svc.Add(ctx, collectorA, draft("qc-2", "Loaded"))
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	loadedEdit := measure("edit (200 queued)", func(ctx context.Context) {
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

	loadedDelete := measure("delete (queue partly drained)", func(ctx context.Context) {
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
// Count the three aggregate queries directly, including those inside the read-only
// transaction. Database-wide transaction statistics are delayed and can hide N+1
// queries executed within a single transaction (issue #91).
func TestStatisticsCostDoesNotGrowWithCollection(t *testing.T) {
	svc := freshTracedService(t)
	ctx := context.Background()

	measure := func() int64 {
		return countQueries(ctx, func(ctx context.Context) {
			if _, err := svc.Statistics(ctx, collectorA); err != nil {
				t.Fatalf("statistics: %v", err)
			}
		})
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

	if small != 3 || large != 3 {
		t.Errorf("statistics cost %d for an empty vault and %d for a 60-item one across 25 series "+
			"and 17 manufacturers; want exactly 3 queries for each (no N+1)", small, large)
	}
}
