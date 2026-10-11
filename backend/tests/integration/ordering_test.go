//go:build integration

package integration

import (
	"context"
	"testing"
)

// T038 — FR-028: editing never moves a collectible in the gallery.
//
// The gallery orders by when entries were added, and an edit writes neither created_at nor the id.
// So this holds by construction — which is exactly why it needs a test. "Order by recently
// updated" is a one-line change that would break it silently, and a collector would experience it
// as their collection rearranging itself whenever they fixed a typo.
func TestEditingDoesNotMoveACollectible(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	names := []string{"First added", "Second added", "Third added"}
	var ids []string
	for i, n := range names {
		added, _, err := svc.Add(ctx, collectorA, draft(string(rune('a'+i)), n))
		if err != nil {
			t.Fatalf("add %s: %v", n, err)
		}
		ids = append(ids, added.Row.Collectible.ID.String())
	}

	before, err := svc.List(ctx, collectorA, nil, "", 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var orderBefore []string
	for _, r := range before.Rows {
		orderBefore = append(orderBefore, r.Collectible.ID.String())
	}

	// Edit the oldest — the one furthest from the front of a newest-first gallery, so a change in
	// position would be unmistakable.
	oldest, err := svc.Get(ctx, collectorA, before.Rows[len(before.Rows)-1].Collectible.ID)
	if err != nil {
		t.Fatalf("read the oldest: %v", err)
	}
	e := edit(oldest.Collectible, oldest.Collectible.Version)
	e.Name = "First added, corrected"
	if _, v, err := svc.Edit(ctx, collectorA, oldest.Collectible.ID, e); err != nil || len(v) > 0 {
		t.Fatalf("edit: %v %+v", err, v)
	}

	after, err := svc.List(ctx, collectorA, nil, "", 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var orderAfter []string
	for _, r := range after.Rows {
		orderAfter = append(orderAfter, r.Collectible.ID.String())
	}

	if len(orderBefore) != len(orderAfter) {
		t.Fatalf("the collection holds %d entries after an edit, had %d", len(orderAfter), len(orderBefore))
	}
	for i := range orderBefore {
		if orderBefore[i] != orderAfter[i] {
			t.Fatalf("the gallery reordered after an edit: %v became %v — a collectible keeps the "+
				"place it has had since it was added (FR-028)", orderBefore, orderAfter)
		}
	}
	_ = ids
}
