//go:build integration

package integration

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
)

// edit builds an EditDraft carrying the attributes a stored collectible already has, so a test can
// change one thing and leave the rest alone — which is what the edit form does.
func edit(c collectible.Collectible, version int) collectible.EditDraft {
	s := collectible.Submitted{Name: c.Name, Status: string(c.Status)}
	str := func(p *string) *string { return p }
	s.Character, s.Series, s.Manufacturer = str(c.Character), str(c.Series), str(c.Manufacturer)
	s.Category, s.Scale, s.Edition, s.Notes = str(c.Category), str(c.Scale), str(c.Edition), str(c.Notes)
	if c.PurchasePrice != nil {
		p := c.PurchasePrice.String()
		s.PurchasePrice = &p
	}
	if c.PurchaseDate != nil {
		p := c.PurchaseDate.Format("2006-01-02")
		s.PurchaseDate = &p
	}
	if c.ReleaseDate != nil {
		p := c.ReleaseDate.Format("2006-01-02")
		s.ReleaseDate = &p
	}
	if c.ImageID != nil {
		p := c.ImageID.String()
		s.ImageID = &p
	}
	return collectible.EditDraft{ExpectedVersion: version, Submitted: s}
}

// T033 — FR-007, FR-008: an edit changes exactly one collectible and creates none.
func TestEditChangesExactlyOneCollectible(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	// Three entries, two of them identical. A change to one must not touch its twin.
	first, _, err := svc.Add(ctx, collectorA, draft("e-1", "Kaiju Sentinel"))
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	twin, _, err := svc.Add(ctx, collectorA, draft("e-2", "Kaiju Sentinel"))
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, _, err := svc.Add(ctx, collectorA, draft("e-3", "Harbour Golem")); err != nil {
		t.Fatalf("add: %v", err)
	}

	target := first.Row.Collectible
	if target.Version != 1 {
		t.Fatalf("a new collectible starts at version %d, want 1", target.Version)
	}

	e := edit(target, target.Version)
	e.Name = "Kaiju Sentinel MkII"
	e.Status = "sold"

	updated, v, err := svc.Edit(ctx, collectorA, target.ID, e)
	if err != nil || len(v) > 0 {
		t.Fatalf("edit: %v %+v", err, v)
	}

	if updated.Collectible.Name != "Kaiju Sentinel MkII" {
		t.Errorf("name = %q, want the edited one", updated.Collectible.Name)
	}
	if updated.Collectible.Status != collectible.StatusSold {
		t.Errorf("status = %q, want sold — any status may become any other (FR-006)", updated.Collectible.Status)
	}
	if updated.Collectible.Version != target.Version+1 {
		t.Errorf("version = %d, want %d — every change raises it (FR-027a)",
			updated.Collectible.Version, target.Version+1)
	}
	if updated.Collectible.ID != target.ID {
		t.Error("the edit returned a different collectible")
	}

	// The twin is untouched, which is the assertion identical entries exist to make (FR-007).
	stillTwin, err := svc.Get(ctx, collectorA, twin.Row.Collectible.ID)
	if err != nil {
		t.Fatalf("read the twin: %v", err)
	}
	if stillTwin.Collectible.Name != "Kaiju Sentinel" {
		t.Errorf("the twin's name became %q; an edit must reach one entry only",
			stillTwin.Collectible.Name)
	}
	if stillTwin.Collectible.Version != 1 {
		t.Errorf("the twin's version moved to %d; it was not edited", stillTwin.Collectible.Version)
	}

	// FR-008: no new collectible.
	page, err := svc.List(ctx, collectorA, nil, "", 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Rows) != 3 {
		t.Errorf("the collection holds %d entries after an edit, want 3 — editing never adds",
			len(page.Rows))
	}
}

// T033 — FR-009: a refused edit leaves the collectible exactly as it was.
func TestARefusedEditChangesNothing(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	d := draft("e-atomic", "Kaiju Sentinel")
	series, price := "Kaiju Wars", "1250.00"
	d.Series, d.PurchasePrice = &series, &price
	added, _, err := svc.Add(ctx, collectorA, d)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	before := added.Row.Collectible

	// One good change and one bad one in the same submission. Nothing may land.
	e := edit(before, before.Version)
	e.Series = strPtr("Kaiju Wars Redux")
	e.PurchasePrice = strPtr("-1.00")

	if _, v, err := svc.Edit(ctx, collectorA, before.ID, e); err != nil {
		t.Fatalf("edit returned an error rather than violations: %v", err)
	} else if len(v) == 0 {
		t.Fatal("a negative price was accepted on an edit; adding refuses it (FR-010)")
	}

	after, err := svc.Get(ctx, collectorA, before.ID)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if after.Collectible.Series == nil || *after.Collectible.Series != "Kaiju Wars" {
		t.Errorf("series = %v after a refused edit; the valid half of an invalid submission must "+
			"not land (FR-009)", after.Collectible.Series)
	}
	if after.Collectible.Version != before.Version {
		t.Errorf("version moved to %d on a refused edit", after.Collectible.Version)
	}
}

// T034 — FR-005: every optional attribute can be cleared back to nothing.
func TestOptionalAttributesCanBeCleared(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	d := draft("e-clear", "Kaiju Sentinel")
	for _, f := range []**string{&d.Character, &d.Series, &d.Manufacturer, &d.Category,
		&d.Scale, &d.Edition, &d.Notes, &d.PurchasePrice, &d.PurchaseDate, &d.ReleaseDate} {
		*f = strPtr("")
	}
	d.Character, d.Series = strPtr("Sentinel Prime"), strPtr("Kaiju Wars")
	d.PurchasePrice, d.PurchaseDate = strPtr("1250.00"), strPtr("2026-08-14")
	d.Notes = strPtr("Box has a dent.")

	added, v, err := svc.Add(ctx, collectorA, d)
	if err != nil || len(v) > 0 {
		t.Fatalf("add: %v %+v", err, v)
	}
	before := added.Row.Collectible

	// Only a name and a status survive — a valid collectible, and the minimum one (FR-005).
	cleared := collectible.EditDraft{
		ExpectedVersion: before.Version,
		Submitted:       collectible.Submitted{Name: before.Name, Status: string(before.Status)},
	}
	updated, v, err := svc.Edit(ctx, collectorA, before.ID, cleared)
	if err != nil || len(v) > 0 {
		t.Fatalf("clearing edit: %v %+v", err, v)
	}

	c := updated.Collectible
	if c.Character != nil || c.Series != nil || c.Notes != nil || c.PurchaseDate != nil {
		t.Errorf("cleared attributes came back set: %+v", c)
	}
	// The one most likely to go wrong: a cleared amount must be absent, not zero (FR-005).
	if c.PurchasePrice != nil {
		t.Errorf("purchase price = %q after being cleared, want absent — \"not recorded\" is not "+
			"the same as 0.00", c.PurchasePrice.String())
	}

	// Confirm against the column itself; a nil in Go could still be a 0.00 in the row.
	var price *string
	if err := pool.QueryRow(ctx,
		`SELECT purchase_price::text FROM collectibles WHERE id = $1`, c.ID).Scan(&price); err != nil {
		t.Fatalf("read the column: %v", err)
	}
	if price != nil {
		t.Errorf("purchase_price is %q in the database, want NULL", *price)
	}
}

// T039 — FR-018: an edit that sends the current imageId back keeps the photograph, and queues
// nothing.
//
// This is the sharpest edge in the contract. A full replacement means an omitted imageId removes
// the photograph; the failure mode is an edit that fixes a typo and silently destroys an image.
func TestEditingOnlyTheNameKeepsThePhotograph(t *testing.T) {
	store, svc := freshStore(t)
	ctx := context.Background()

	img, err := svc.UploadImage(ctx, collectorA, bytes.NewReader(testJPEG(t, 300, 400)))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	imageID := img.ID
	d := draft("e-image", "Kaiju Sentinel")
	d.ImageID = strPtr(imageID.String())
	added, v, err := svc.Add(ctx, collectorA, d)
	if err != nil || len(v) > 0 {
		t.Fatalf("add with an image: %v %+v", err, v)
	}
	before := added.Row.Collectible
	if before.ImageID == nil {
		t.Fatal("the fixture did not attach an image")
	}

	e := edit(before, before.Version) // carries the current imageId, as the form does
	e.Name = "Kaiju Sentinel MkII"

	updated, v, err := svc.Edit(ctx, collectorA, before.ID, e)
	if err != nil || len(v) > 0 {
		t.Fatalf("edit: %v %+v", err, v)
	}
	if updated.Collectible.ImageID == nil || *updated.Collectible.ImageID != imageID {
		t.Fatalf("the photograph was lost by an edit that never mentioned it (FR-018): %v",
			updated.Collectible.ImageID)
	}

	// The image row survives, so the rendition is still fetchable by its owner.
	if _, err := store.GetImage(ctx, collectorA, imageID); err != nil {
		t.Errorf("the image row was deleted by an edit that kept the photograph: %v", err)
	}
	// And nothing was queued for destruction.
	if n := queueDepth(t); n != 0 {
		t.Errorf("%d image deletions queued by an edit that changed only the name", n)
	}
}

// T036 — FR-030, FR-031: another collector's collectible is unreachable, and indistinguishable
// from one that never existed.
func TestEditingAnotherCollectorsCollectible(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	added, _, err := svc.Add(ctx, collectorA, draft("e-mine", "Private Statue"))
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	mine := added.Row.Collectible

	t.Run("reading", func(t *testing.T) {
		_, theirErr := svc.Get(ctx, collectorB, mine.ID)
		_, fictionErr := svc.Get(ctx, collectorB, uuid.New())
		if !errors.Is(theirErr, postgres.ErrNotFound) {
			t.Errorf("reading another collector's collectible gave %v, want not-found", theirErr)
		}
		if !errors.Is(fictionErr, postgres.ErrNotFound) {
			t.Errorf("reading a fictional id gave %v, want not-found", fictionErr)
		}
	})

	t.Run("editing", func(t *testing.T) {
		e := edit(mine, mine.Version)
		e.Name = "Taken"
		_, _, err := svc.Edit(ctx, collectorB, mine.ID, e)
		if !errors.Is(err, postgres.ErrNotFound) {
			t.Fatalf("editing another collector's collectible gave %v, want not-found — and never "+
				"a distinct refusal, which would confirm it exists (FR-031)", err)
		}

		// Nothing changed, which is the part a status code cannot demonstrate.
		after, err := svc.Get(ctx, collectorA, mine.ID)
		if err != nil {
			t.Fatalf("read back: %v", err)
		}
		if after.Collectible.Name != "Private Statue" || after.Collectible.Version != mine.Version {
			t.Errorf("collector B's refused edit still changed something: %+v", after.Collectible)
		}
	})
}

func strPtr(s string) *string { return &s }
