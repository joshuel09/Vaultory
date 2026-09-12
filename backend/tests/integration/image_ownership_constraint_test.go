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
