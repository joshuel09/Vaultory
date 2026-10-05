//go:build integration

package integration

import (
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
