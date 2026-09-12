//go:build integration

package integration

import (
	"context"
	"sync"
	"testing"
)

// FR-047, SC-016: replaying a submission key returns the collectible it already created.
//
// This is what stops an accidental double-click from stranding a collector with a duplicate they
// cannot delete, since deleting is out of scope for this feature.
func TestReplayedSubmissionKeyReturnsTheSameCollectible(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	d := draft("retry-key", "Retry Test")

	first, v, err := svc.Add(ctx, collectorA, d)
	if err != nil || len(v) > 0 {
		t.Fatalf("first add: %v %+v", err, v)
	}
	if first.Existing {
		t.Error("the first add should not be reported as a replay")
	}

	second, v, err := svc.Add(ctx, collectorA, d)
	if err != nil || len(v) > 0 {
		t.Fatalf("replay: %v %+v", err, v)
	}
	if !second.Existing {
		t.Error("a replayed key should be reported as existing")
	}
	if first.Row.Collectible.ID != second.Row.Collectible.ID {
		t.Fatalf("replay created a second collectible (%s vs %s); FR-047 requires exactly one",
			first.Row.Collectible.ID, second.Row.Collectible.ID)
	}

	page, err := svc.List(ctx, collectorA, nil, "", 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Rows) != 1 {
		t.Errorf("collection has %d entries after a replay, want 1", len(page.Rows))
	}
}

// The decisive case: simultaneous requests carrying one key.
//
// A check-then-insert would let several through here — both would find nothing and both would
// write. Only the uniqueness constraint on (collector_id, submission_key) actually holds, which is
// why the schema carries it.
func TestConcurrentSubmissionsWithOneKeyCreateOneCollectible(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	const attempts = 8
	d := draft("concurrent-key", "Concurrent Test")

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		ids  = map[string]int{}
		errs []error
	)
	wg.Add(attempts)
	for i := 0; i < attempts; i++ {
		go func() {
			defer wg.Done()
			res, v, err := svc.Add(ctx, collectorA, d)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			if len(v) > 0 {
				t.Errorf("unexpected violations: %+v", v)
				return
			}
			ids[res.Row.Collectible.ID.String()]++
		}()
	}
	wg.Wait()

	for _, err := range errs {
		t.Errorf("concurrent add failed: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("%d distinct collectibles were created from one submission key, want 1: %v", len(ids), ids)
	}

	page, err := svc.List(ctx, collectorA, nil, "", 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Rows) != 1 {
		t.Errorf("collection has %d entries after %d concurrent submissions, want 1", len(page.Rows), attempts)
	}
}

// A key belongs to one collector. Two collectors using the same key each get their own
// collectible — the primary key is composite for exactly this reason.
func TestSubmissionKeysAreScopedToTheCollector(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	d := draft("shared-key", "Same Key")

	a, _, err := svc.Add(ctx, collectorA, d)
	if err != nil {
		t.Fatalf("add for A: %v", err)
	}
	b, _, err := svc.Add(ctx, collectorB, d)
	if err != nil {
		t.Fatalf("add for B: %v", err)
	}
	if a.Row.Collectible.ID == b.Row.Collectible.ID {
		t.Fatal("two collectors sharing a submission key received the same collectible")
	}
	if b.Existing {
		t.Error("B's add was treated as a replay of A's")
	}
}
