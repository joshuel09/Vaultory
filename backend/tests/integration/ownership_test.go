//go:build integration

package integration

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
)

func testJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 90, G: 90, B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

func draft(key, name string) collectible.Draft {
	return collectible.Draft{SubmissionKey: key, Name: name, Status: "owned"}
}

// FR-026: one collector's collection never contains another's collectibles.
func TestCollectionsAreIsolatedBetweenCollectors(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	if _, v, err := svc.Add(ctx, collectorA, draft("a-1", "A's Sentinel")); err != nil || len(v) > 0 {
		t.Fatalf("add for A: %v %+v", err, v)
	}
	if _, v, err := svc.Add(ctx, collectorB, draft("b-1", "B's Guardian")); err != nil || len(v) > 0 {
		t.Fatalf("add for B: %v %+v", err, v)
	}

	pageA, err := svc.List(ctx, collectorA, nil, "", 50)
	if err != nil {
		t.Fatalf("list A: %v", err)
	}
	if len(pageA.Rows) != 1 || pageA.Rows[0].Collectible.Name != "A's Sentinel" {
		t.Fatalf("A sees %d collectibles, want only their own", len(pageA.Rows))
	}
	if pageA.TotalUnfiltered != 1 {
		t.Errorf("A's total = %d, want 1 — the count must be scoped too", pageA.TotalUnfiltered)
	}

	pageB, err := svc.List(ctx, collectorB, nil, "", 50)
	if err != nil {
		t.Fatalf("list B: %v", err)
	}
	if len(pageB.Rows) != 1 || pageB.Rows[0].Collectible.Name != "B's Guardian" {
		t.Fatalf("B sees %d collectibles, want only their own", len(pageB.Rows))
	}
}

// FR-015, FR-027: another collector's image is not readable, and the refusal is "not found" rather
// than "forbidden" — a 403 would confirm the image exists.
func TestAnotherCollectorsImageIsNotFound(t *testing.T) {
	store, svc := freshStore(t)
	ctx := context.Background()

	img, err := svc.UploadImage(ctx, collectorA, bytes.NewReader(testJPEG(t, 600, 800)))
	if err != nil {
		t.Fatalf("upload for A: %v", err)
	}

	// The owner can read it.
	rc, err := svc.OpenRendition(ctx, collectorA, img.ID)
	if err != nil {
		t.Fatalf("owner cannot read their own rendition: %v", err)
	}
	_ = rc.Close()

	// The other collector cannot, and gets the same answer as for an image that does not exist.
	_, err = svc.OpenRendition(ctx, collectorB, img.ID)
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("B reading A's rendition: %v, want ErrNotFound (FR-015, FR-027)", err)
	}
	_, err = svc.OpenRendition(ctx, collectorB, uuid.New())
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("B reading a nonexistent rendition: %v, want the identical ErrNotFound", err)
	}

	if _, err := store.GetImage(ctx, collectorB, img.ID); !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("store.GetImage across collectors: %v, want ErrNotFound", err)
	}
}

// FR-015: a collectible may not reference another collector's image. Rejected with a field-level
// message, and — see image_ownership_constraint_test.go — by the database itself.
func TestCannotAttachAnotherCollectorsImage(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	img, err := svc.UploadImage(ctx, collectorA, bytes.NewReader(testJPEG(t, 400, 500)))
	if err != nil {
		t.Fatalf("upload for A: %v", err)
	}

	id := img.ID.String()
	d := draft("b-steal", "Borrowed Image")
	d.ImageID = &id

	_, violations, err := svc.Add(ctx, collectorB, d)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if len(violations) != 1 || violations[0].Field != "imageId" {
		t.Fatalf("violations = %+v, want one on imageId", violations)
	}
	// The message must not distinguish "someone else's" from "does not exist" (FR-027).
	if violations[0].Message != "That image could not be found." {
		t.Errorf("message %q reveals more than it should", violations[0].Message)
	}
}
