//go:build integration

package integration

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/joshuel09/vaultory/backend/internal/collection"
	"github.com/joshuel09/vaultory/backend/internal/imaging"
)

// FR-013: a refused image leaves nothing behind, and does not stop the collectible from being
// saved without one.
func TestRefusedUploadLeavesNothingAndDoesNotBlockSaving(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	countImages := func() int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM collectible_images`).Scan(&n); err != nil {
			t.Fatalf("count images: %v", err)
		}
		return n
	}

	// Oversized (FR-010).
	oversized := bytes.Repeat([]byte{0x7F}, int(imaging.MaxUploadBytes)+1)
	if _, err := svc.UploadImage(ctx, collectorA, bytes.NewReader(oversized)); !errors.Is(err, collection.ErrImageTooLarge) {
		t.Errorf("oversized upload: %v, want ErrImageTooLarge", err)
	}
	if n := countImages(); n != 0 {
		t.Errorf("%d image rows after a refused oversized upload, want 0", n)
	}

	// Wrong format, declared as an image (FR-009). A naive Content-Type check would let this pass.
	pdf := []byte("%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\n")
	if _, err := svc.UploadImage(ctx, collectorA, bytes.NewReader(pdf)); !errors.Is(err, collection.ErrImageUnsupported) {
		t.Errorf("non-image upload: %v, want ErrImageUnsupported", err)
	}
	if n := countImages(); n != 0 {
		t.Errorf("%d image rows after a refused non-image upload, want 0", n)
	}

	// The collectible still saves, with no image.
	res, v, err := svc.Add(ctx, collectorA, draft("after-refusal", "Saved Without Image"))
	if err != nil || len(v) > 0 {
		t.Fatalf("add after a refused upload: %v %+v", err, v)
	}
	if res.Row.Collectible.ImageID != nil {
		t.Error("the collectible should have no image")
	}

	// And a good upload still works afterwards.
	img, err := svc.UploadImage(ctx, collectorA, bytes.NewReader(testJPEG(t, 500, 600)))
	if err != nil {
		t.Fatalf("valid upload after refusals: %v", err)
	}
	if img.ContentType != "image/jpeg" {
		t.Errorf("content type = %q, want image/jpeg", img.ContentType)
	}
	if img.Width != 500 || img.Height != 600 {
		t.Errorf("original dimensions = %dx%d, want 500x600", img.Width, img.Height)
	}
	if n := countImages(); n != 1 {
		t.Errorf("%d image rows after one good upload, want 1", n)
	}
}
