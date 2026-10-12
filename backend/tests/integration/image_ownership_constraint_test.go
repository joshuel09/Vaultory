//go:build integration

package integration

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// FR-015 at the schema level.
//
// The application already refuses this, but application checks can be bypassed by the next code
// path someone writes. This asserts the composite foreign key on (image_id, collector_id) — the
// guarantee that does not depend on anybody remembering.
func TestDatabaseRefusesCrossCollectorImageReference(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	img, err := svc.UploadImage(ctx, collectorA, bytes.NewReader(testJPEG(t, 300, 400)))
	if err != nil {
		t.Fatalf("upload for A: %v", err)
	}
	res, _, err := svc.Add(ctx, collectorB, draft("b-plain", "B's own"))
	if err != nil {
		t.Fatalf("add for B: %v", err)
	}

	// Go straight to SQL, bypassing every application check, and try to point B's collectible at
	// A's image. The composite key must refuse it.
	_, err = pool.Exec(ctx,
		`UPDATE collectibles SET image_id = $1 WHERE id = $2`,
		img.ID, res.Row.Collectible.ID)
	if err == nil {
		t.Fatal("the database accepted a collectible referencing another collector's image; " +
			"the composite foreign key on (image_id, collector_id) is missing or wrong (FR-015)")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "foreign key") {
		t.Errorf("refused with %v, expected a foreign key violation", err)
	}

	// The same reference is fine for the image's own collector.
	resA, _, err := svc.Add(ctx, collectorA, draft("a-plain", "A's own"))
	if err != nil {
		t.Fatalf("add for A: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`UPDATE collectibles SET image_id = $1 WHERE id = $2`,
		img.ID, resA.Row.Collectible.ID); err != nil {
		t.Errorf("the owning collector should be able to reference their own image: %v", err)
	}
}

// T067 — FR-021: an edit cannot point a collectible at another collector's image, and the
// database refuses it even when the application check is bypassed.
//
// Editing is the first operation that can *change* image_id on a row that already exists. The
// composite foreign key covered inserting; this covers updating, which is a different statement
// and could have been written without the constraint applying.
func TestEditCannotBorrowAnotherCollectorsImage(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	theirs, err := svc.UploadImage(ctx, collectorB, bytes.NewReader(testJPEG(t, 300, 400)))
	if err != nil {
		t.Fatalf("upload for B: %v", err)
	}
	added, _, err := svc.Add(ctx, collectorA, draft("borrow", "A's own"))
	if err != nil {
		t.Fatalf("add for A: %v", err)
	}
	mine := added.Row.Collectible

	t.Run("the application refuses it as an unknown image", func(t *testing.T) {
		e := edit(mine, mine.Version)
		e.ImageID = strPtr(theirs.ID.String())

		_, violations, err := svc.Edit(ctx, collectorA, mine.ID, e)
		if err != nil {
			t.Fatalf("edit: %v", err)
		}
		if len(violations) == 0 {
			t.Fatal("A was allowed to reference B's image (FR-021)")
		}
		// The same words an image that never existed would get. Telling A that the image exists
		// but is not theirs would disclose B's collection (FR-031).
		if violations[0].Field != "imageId" || !strings.Contains(violations[0].Message, "could not be found") {
			t.Errorf("violation = %+v, want an indistinguishable unknown-image message", violations[0])
		}
	})

	t.Run("the database refuses it too", func(t *testing.T) {
		// Straight to SQL, past every application check — the guarantee that does not depend on
		// anybody remembering.
		_, err := pool.Exec(ctx,
			`UPDATE collectibles SET image_id = $1 WHERE id = $2`, theirs.ID, mine.ID)
		if err == nil {
			t.Fatal("an UPDATE set a collectible's image to another collector's; the composite " +
				"foreign key does not cover updates (FR-021)")
		}
		if !strings.Contains(strings.ToLower(err.Error()), "foreign key") {
			t.Errorf("refused with %v, expected a foreign key violation", err)
		}
	})
}
