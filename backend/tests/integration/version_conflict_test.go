//go:build integration

package integration

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
)

// T035 — FR-027: a save made from a view that is no longer current is refused, not applied.
//
// Principle IV forbids collection information being "silently lost, overwritten, or corrupted".
// Last-write-wins is precisely that: the first collector's correction disappears and nobody is
// told. This is the test that makes the refusal real.
func TestAStaleEditIsRefused(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	added, _, err := svc.Add(ctx, collectorA, draft("vc-1", "Kaiju Sentinel"))
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	opened := added.Row.Collectible // what both tabs were shown

	// Tab A saves first.
	firstEdit := edit(opened, opened.Version)
	firstEdit.Name = "Kaiju Sentinel MkII"
	winner, v, err := svc.Edit(ctx, collectorA, opened.ID, firstEdit)
	if err != nil || len(v) > 0 {
		t.Fatalf("the first edit should have succeeded: %v %+v", err, v)
	}

	// Tab B saves second, from the version it was opened at.
	secondEdit := edit(opened, opened.Version)
	secondEdit.Notes = strPtr("A note written in the other tab")
	_, _, err = svc.Edit(ctx, collectorA, opened.ID, secondEdit)

	var conflict *postgres.VersionConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("the stale edit returned %v, want a version conflict — accepting it would "+
			"silently discard the first edit (FR-027)", err)
	}

	// The refusal carries the collectible as it now stands, so the collector can be shown what it
	// actually says without a second request.
	if conflict.Current.Collectible.Name != "Kaiju Sentinel MkII" {
		t.Errorf("the conflict reported name %q, want the winning edit's",
			conflict.Current.Collectible.Name)
	}
	if conflict.Current.Collectible.Version != winner.Collectible.Version {
		t.Errorf("the conflict reported version %d, want %d",
			conflict.Current.Collectible.Version, winner.Collectible.Version)
	}

	// And nothing of the losing edit landed.
	after, err := svc.Get(ctx, collectorA, opened.ID)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if after.Collectible.Notes != nil {
		t.Errorf("the refused edit's notes were written anyway: %q", *after.Collectible.Notes)
	}
	if after.Collectible.Name != "Kaiju Sentinel MkII" {
		t.Errorf("name = %q; the winning edit was overwritten", after.Collectible.Name)
	}
}

// Two edits arriving at once produce one winner and one refusal — never two winners.
//
// The sequential case above would pass with a plain compare-then-write. This one is what SELECT …
// FOR UPDATE is for: without the lock, both transactions read version 1, both find it current, and
// both write.
func TestConcurrentEditsProduceOneWinner(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	added, _, err := svc.Add(ctx, collectorA, draft("vc-2", "Kaiju Sentinel"))
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	opened := added.Row.Collectible

	const racers = 6
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
		conflicts int
		others    []error
	)
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			e := edit(opened, opened.Version)
			e.Notes = strPtr(string(rune('A' + n)))
			_, v, err := svc.Edit(ctx, collectorA, opened.ID, e)

			mu.Lock()
			defer mu.Unlock()
			var conflict *postgres.VersionConflictError
			switch {
			case err == nil && len(v) == 0:
				succeeded++
			case errors.As(err, &conflict):
				conflicts++
			default:
				others = append(others, err)
			}
		}(i)
	}
	wg.Wait()

	if len(others) > 0 {
		t.Fatalf("unexpected failures: %v", others)
	}
	if succeeded != 1 {
		t.Errorf("%d of %d concurrent edits succeeded, want exactly 1 — the rest read the same "+
			"version and must not all win", succeeded, racers)
	}
	if conflicts != racers-1 {
		t.Errorf("%d conflicts, want %d", conflicts, racers-1)
	}

	// One winner means one increment.
	after, err := svc.Get(ctx, collectorA, opened.ID)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if after.Collectible.Version != opened.Version+1 {
		t.Errorf("version = %d after %d concurrent edits, want %d",
			after.Collectible.Version, racers, opened.Version+1)
	}
}
