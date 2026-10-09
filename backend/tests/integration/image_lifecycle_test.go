//go:build integration

package integration

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/collection"
	"github.com/joshuel09/vaultory/backend/internal/imagestore"
	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
)

// queueOne writes an image's files and a queue entry describing them, which is the state the
// database is left in once an edit or a deletion has released an image.
func queueOne(t *testing.T, images imagestore.Store, collector uuid.UUID) (uuid.UUID, string, string) {
	t.Helper()
	ctx := context.Background()
	id := uuid.New()
	originalKey := "orig/" + id.String()
	renditionKey := "rend/" + id.String()

	for _, key := range []string{originalKey, renditionKey} {
		if err := images.Put(ctx, key, []byte("not really an image, but bytes on disk")); err != nil {
			t.Fatalf("put %s: %v", key, err)
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO pending_image_deletions (image_id, collector_id, original_key, rendition_key)
		VALUES ($1, $2, $3, $4)`, id, collector, originalKey, renditionKey); err != nil {
		t.Fatalf("queue a deletion: %v", err)
	}
	return id, originalKey, renditionKey
}

func queueDepth(t *testing.T) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM pending_image_deletions`).Scan(&n); err != nil {
		t.Fatalf("count queue: %v", err)
	}
	return n
}

func stillStored(t *testing.T, images imagestore.Store, key string) bool {
	t.Helper()
	rc, err := images.Open(context.Background(), key)
	if errors.Is(err, imagestore.ErrNotFound) {
		return false
	}
	if err != nil {
		t.Fatalf("open %s: %v", key, err)
	}
	_ = rc.Close()
	return true
}

// T031: the drain removes the files of images whose rows are already gone, and clears the queue.
func TestDrainRemovesOrphanedImageFiles(t *testing.T) {
	_, _ = freshStore(t)
	dir := t.TempDir()
	images, err := imagestore.NewFilesystem(dir)
	if err != nil {
		t.Fatalf("image store: %v", err)
	}
	svc := collection.NewService(postgres.NewStore(pool), images, time.Hour)

	_, originalKey, renditionKey := queueOne(t, images, collectorA)
	if !stillStored(t, images, originalKey) || !stillStored(t, images, renditionKey) {
		t.Fatal("the fixture did not write both files")
	}

	if drained := svc.DrainImageDeletions(context.Background()); drained != 1 {
		t.Errorf("drained %d images, want 1", drained)
	}
	if stillStored(t, images, originalKey) {
		t.Error("the original is still on disk after a drain")
	}
	if stillStored(t, images, renditionKey) {
		t.Error("the rendition is still on disk after a drain")
	}
	if n := queueDepth(t); n != 0 {
		t.Errorf("queue holds %d entries after a successful drain, want 0", n)
	}
}

// failingStore refuses to delete anything. Everything else behaves.
type failingStore struct {
	imagestore.Store
	failures int
}

func (f *failingStore) Delete(context.Context, string) error {
	f.failures++
	return errors.New("storage is unavailable")
}

func (f *failingStore) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	return f.Store.Open(ctx, key)
}

// FR-020a: a file that could not be destroyed is retried rather than abandoned.
//
// This is the property that lets the file deletion sit outside the transaction. If a failure lost
// the entry, a storage blip would leak bytes permanently — and the alternative, failing the
// collector's deletion, would let a storage fault make a collectible undeletable.
func TestAFailedDrainKeepsItsQueueEntry(t *testing.T) {
	_, _ = freshStore(t)
	dir := t.TempDir()
	real, err := imagestore.NewFilesystem(dir)
	if err != nil {
		t.Fatalf("image store: %v", err)
	}
	broken := &failingStore{Store: real}

	_, originalKey, _ := queueOne(t, real, collectorA)

	svc := collection.NewService(postgres.NewStore(pool), broken, time.Hour)
	if drained := svc.DrainImageDeletions(context.Background()); drained != 0 {
		t.Errorf("drained %d with a broken store, want 0", drained)
	}
	if broken.failures == 0 {
		t.Error("the drain did not attempt a deletion at all")
	}
	if n := queueDepth(t); n != 1 {
		t.Fatalf("queue holds %d entries after a failed drain, want 1 — the intent to delete must "+
			"outlive the failure (FR-020a)", n)
	}
	if !stillStored(t, real, originalKey) {
		t.Error("the file is gone even though the store reported failure")
	}

	// The retry: a working store on a later request finishes the job.
	working := collection.NewService(postgres.NewStore(pool), real, time.Hour)
	if drained := working.DrainImageDeletions(context.Background()); drained != 1 {
		t.Error("the entry left by the failure was not picked up by a later drain")
	}
	if n := queueDepth(t); n != 0 {
		t.Errorf("queue holds %d entries after the retry succeeded, want 0", n)
	}
	if stillStored(t, real, originalKey) {
		t.Error("the file survived a successful retry")
	}
}

// An absent file counts as deleted, so a half-finished earlier attempt cannot wedge the queue.
func TestDrainToleratesAFileThatIsAlreadyGone(t *testing.T) {
	_, _ = freshStore(t)
	images, err := imagestore.NewFilesystem(t.TempDir())
	if err != nil {
		t.Fatalf("image store: %v", err)
	}
	svc := collection.NewService(postgres.NewStore(pool), images, time.Hour)

	_, originalKey, _ := queueOne(t, images, collectorA)
	// Simulate a drain that deleted one file and then died before clearing the entry.
	if err := images.Delete(context.Background(), originalKey); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if drained := svc.DrainImageDeletions(context.Background()); drained != 1 {
		t.Error("a partly-deleted entry was not finished off")
	}
	if n := queueDepth(t); n != 0 {
		t.Errorf("queue holds %d entries, want 0", n)
	}
}

// attach uploads an image and returns a collectible referencing it.
func attach(t *testing.T, svc *collection.Service, key string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	img, err := svc.UploadImage(ctx, collectorA, bytes.NewReader(testJPEG(t, 300, 400)))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	d := draft(key, "Kaiju Sentinel")
	d.ImageID = strPtr(img.ID.String())
	added, v, err := svc.Add(ctx, collectorA, d)
	if err != nil || len(v) > 0 {
		t.Fatalf("add with an image: %v %+v", err, v)
	}
	return added.Row.Collectible.ID, img.ID
}

// T065 — FR-020: replacing a photograph makes the old one unretrievable at once, and leaves the
// new one alone.
func TestReplacingAPhotographReleasesTheOldOne(t *testing.T) {
	store, svc := freshStore(t)
	ctx := context.Background()

	collectibleID, oldImage := attach(t, svc, "img-replace")

	replacement, err := svc.UploadImage(ctx, collectorA, bytes.NewReader(testJPEG(t, 400, 500)))
	if err != nil {
		t.Fatalf("upload the replacement: %v", err)
	}

	current, err := svc.Get(ctx, collectorA, collectibleID)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	e := edit(current.Collectible, current.Collectible.Version)
	e.ImageID = strPtr(replacement.ID.String())

	updated, v, err := svc.Edit(ctx, collectorA, collectibleID, e)
	if err != nil || len(v) > 0 {
		t.Fatalf("edit: %v %+v", err, v)
	}
	if updated.Collectible.ImageID == nil || *updated.Collectible.ImageID != replacement.ID {
		t.Fatalf("the collectible points at %v, want the replacement", updated.Collectible.ImageID)
	}

	// The old photograph is gone, by its owner's own session. Deleting the image row is what does
	// this — unlinking would not, because a rendition is authorized on the image's own
	// collector_id, so that it can be previewed before any collectible references it.
	if _, err := svc.OpenRendition(ctx, collectorA, oldImage); err == nil {
		t.Error("the replaced photograph is still readable by its owner (FR-020)")
	}
	if _, err := store.GetImage(ctx, collectorA, oldImage); err == nil {
		t.Error("the replaced image's row survived")
	}

	// The new one is untouched and still readable.
	if rc, err := svc.OpenRendition(ctx, collectorA, replacement.ID); err != nil {
		t.Errorf("the replacement is not readable: %v", err)
	} else {
		_ = rc.Close()
	}
}

// T066 — FR-017: omitting the image removes the photograph, and the collectible falls back to the
// designed placeholder.
func TestRemovingAPhotographClearsTheReference(t *testing.T) {
	store, svc := freshStore(t)
	ctx := context.Background()

	collectibleID, imageID := attach(t, svc, "img-remove")

	current, err := svc.Get(ctx, collectorA, collectibleID)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	e := edit(current.Collectible, current.Collectible.Version)
	e.ImageID = nil // the full replacement's meaning: this collectible has no photograph

	updated, v, err := svc.Edit(ctx, collectorA, collectibleID, e)
	if err != nil || len(v) > 0 {
		t.Fatalf("edit: %v %+v", err, v)
	}
	if updated.Collectible.ImageID != nil {
		t.Errorf("image_id = %v after removal, want NULL", updated.Collectible.ImageID)
	}
	if updated.RenditionKey != nil {
		t.Error("a rendition key came back for a collectible with no photograph; the gallery " +
			"would try to render one instead of the placeholder (FR-017)")
	}
	if _, err := store.GetImage(ctx, collectorA, imageID); err == nil {
		t.Error("the removed image's row survived, so it is still fetchable by its owner (FR-020)")
	}

	// Confirm against the column, not just the Go value.
	var stored *uuid.UUID
	if err := pool.QueryRow(ctx,
		`SELECT image_id FROM collectibles WHERE id = $1`, collectibleID).Scan(&stored); err != nil {
		t.Fatalf("read the column: %v", err)
	}
	if stored != nil {
		t.Errorf("image_id is %v in the database, want NULL", stored)
	}
}

// T068 — FR-019: a refused edit leaves the existing photograph exactly where it was, and queues
// nothing for destruction.
//
// The dangerous shape would be releasing the old image before the edit is known to succeed: a
// collector whose save is rejected for a negative price would lose their photograph to a
// validation error.
func TestARefusedEditKeepsThePhotograph(t *testing.T) {
	store, svc := freshStore(t)
	ctx := context.Background()

	collectibleID, imageID := attach(t, svc, "img-refused")

	t.Run("a validation failure", func(t *testing.T) {
		current, err := svc.Get(ctx, collectorA, collectibleID)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		e := edit(current.Collectible, current.Collectible.Version)
		e.ImageID = nil // they also tried to remove the photograph
		e.PurchasePrice = strPtr("-5.00")

		if _, v, err := svc.Edit(ctx, collectorA, collectibleID, e); err != nil {
			t.Fatalf("edit: %v", err)
		} else if len(v) == 0 {
			t.Fatal("a negative price was accepted")
		}

		after, err := svc.Get(ctx, collectorA, collectibleID)
		if err != nil {
			t.Fatalf("read back: %v", err)
		}
		if after.Collectible.ImageID == nil || *after.Collectible.ImageID != imageID {
			t.Error("a refused edit removed the photograph anyway (FR-019)")
		}
		if _, err := store.GetImage(ctx, collectorA, imageID); err != nil {
			t.Errorf("the image row was deleted by a refused edit: %v", err)
		}
		if n := queueDepth(t); n != 0 {
			t.Errorf("%d deletions queued by a refused edit", n)
		}
	})

	t.Run("a version conflict", func(t *testing.T) {
		current, err := svc.Get(ctx, collectorA, collectibleID)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		opened := current.Collectible

		// Somebody else edits first.
		winner := edit(opened, opened.Version)
		winner.Name = "Won the race"
		if _, v, err := svc.Edit(ctx, collectorA, collectibleID, winner); err != nil || len(v) > 0 {
			t.Fatalf("the first edit failed: %v %+v", err, v)
		}

		// The stale save, which also wanted the photograph gone.
		stale := edit(opened, opened.Version)
		stale.ImageID = nil
		if _, _, err := svc.Edit(ctx, collectorA, collectibleID, stale); err == nil {
			t.Fatal("the stale edit was accepted")
		}

		after, err := svc.Get(ctx, collectorA, collectibleID)
		if err != nil {
			t.Fatalf("read back: %v", err)
		}
		if after.Collectible.ImageID == nil || *after.Collectible.ImageID != imageID {
			t.Error("a refused edit removed the photograph anyway (FR-019)")
		}
		if n := queueDepth(t); n != 0 {
			t.Errorf("%d deletions queued by a refused edit", n)
		}
	})
}
