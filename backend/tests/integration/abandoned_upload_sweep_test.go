//go:build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/collection"
	"github.com/joshuel09/vaultory/backend/internal/imagestore"
	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
)

// Feature 009: uploads nobody attached are swept once they are older than the grace period, and a
// collectible being saved never loses its photograph to the sweep.

// grace mirrors abandonedUploadGrace in the collection package. The tests age uploads either side
// of it rather than waiting thirty days.
const grace = 30 * 24 * time.Hour

// sweepService returns a service over a filesystem image store the test can inspect.
func sweepService(t *testing.T, images imagestore.Store) *collection.Service {
	t.Helper()
	svc := collection.NewService(postgres.NewStore(pool), images, 24*time.Hour)
	svc.SetClock(func() time.Time { return time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC) })
	return svc
}

func newImageStore(t *testing.T) imagestore.Store {
	t.Helper()
	images, err := imagestore.NewFilesystem(t.TempDir())
	if err != nil {
		t.Fatalf("image store: %v", err)
	}
	return images
}

func uploadAs(t *testing.T, svc *collection.Service, collector uuid.UUID) postgres.Image {
	t.Helper()
	img, err := svc.UploadImage(context.Background(), collector, bytes.NewReader(testJPEG(t, 300, 400)))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	return img
}

// age backdates an upload, which is how a test gets one past the grace period without waiting.
func age(t *testing.T, imageID uuid.UUID, by time.Duration) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`UPDATE collectible_images SET created_at = now() - make_interval(secs => $2) WHERE id = $1`,
		imageID, by.Seconds()); err != nil {
		t.Fatalf("age upload: %v", err)
	}
}

func imageRowExists(t *testing.T, imageID uuid.UUID) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(),
		`SELECT EXISTS (SELECT 1 FROM collectible_images WHERE id = $1)`, imageID).Scan(&exists); err != nil {
		t.Fatalf("check image row: %v", err)
	}
	return exists
}

// T006 — US1/AC1: an upload nobody attached, past the grace period, loses its row and both files.
func TestSweepRemovesAnAbandonedUploadAndItsFiles(t *testing.T) {
	_, _ = freshStore(t)
	ctx := context.Background()
	images := newImageStore(t)
	svc := sweepService(t, images)

	img := uploadAs(t, svc, collectorA)
	age(t, img.ID, grace+time.Hour)

	if swept := svc.SweepAbandonedUploads(ctx); swept != 1 {
		t.Fatalf("swept %d uploads, want 1", swept)
	}
	if imageRowExists(t, img.ID) {
		t.Error("the upload's row survived the sweep")
	}
	if n := queueDepth(t); n != 1 {
		t.Fatalf("queue holds %d entries, want 1 — the files must go through 006's queue (FR-003)", n)
	}

	if drained := svc.DrainImageDeletions(ctx); drained != 1 {
		t.Errorf("drained %d, want 1", drained)
	}
	if stillStored(t, images, img.OriginalKey) || stillStored(t, images, img.RenditionKey) {
		t.Error("the upload's files are still stored after the drain")
	}
	if _, err := svc.OpenRendition(ctx, collectorA, img.ID); err == nil {
		t.Error("a swept upload's preview is still served")
	}
}

// T006 — US1/AC2: two photographs chosen in one form, saved with the second. The first goes; the
// saved one stays, however old it is.
func TestSweepTakesTheReplacedUploadAndLeavesTheSavedOne(t *testing.T) {
	_, _ = freshStore(t)
	ctx := context.Background()
	images := newImageStore(t)
	svc := sweepService(t, images)

	first := uploadAs(t, svc, collectorA)
	second := uploadAs(t, svc, collectorA)
	d := draft("sweep-replaced", "Kaiju Sentinel")
	d.ImageID = strPtr(second.ID.String())
	added, v, err := svc.Add(ctx, collectorA, d)
	if err != nil || len(v) > 0 {
		t.Fatalf("add: %v %+v", err, v)
	}
	age(t, first.ID, grace+time.Hour)
	age(t, second.ID, grace+time.Hour)

	if swept := svc.SweepAbandonedUploads(ctx); swept != 1 {
		t.Fatalf("swept %d uploads, want exactly the unattached one", swept)
	}
	if imageRowExists(t, first.ID) {
		t.Error("the replaced upload survived")
	}
	if !imageRowExists(t, second.ID) {
		t.Fatal("the saved collectible's photograph was swept (FR-004)")
	}
	row, err := svc.Get(ctx, collectorA, added.Row.Collectible.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if row.Collectible.ImageID == nil || *row.Collectible.ImageID != second.ID {
		t.Errorf("the saved collectible's photograph changed: %v", row.Collectible.ImageID)
	}
}

// T007 — US1/AC3, SC-004: a storage fault during the sweep's drain leaves the files queued, and a
// later drain with working storage removes them.
func TestASweptUploadsFilesAreRetriedAfterAStorageFault(t *testing.T) {
	_, _ = freshStore(t)
	ctx := context.Background()
	real := newImageStore(t)
	img := uploadAs(t, sweepService(t, real), collectorA)
	age(t, img.ID, grace+time.Hour)

	broken := &failingStore{Store: real}
	brokenSvc := sweepService(t, broken)
	if swept := brokenSvc.SweepAbandonedUploads(ctx); swept != 1 {
		t.Fatalf("swept %d, want 1 — the row goes whatever the storage does", swept)
	}
	if drained := brokenSvc.DrainImageDeletions(ctx); drained != 0 {
		t.Errorf("drained %d with broken storage, want 0", drained)
	}
	if n := queueDepth(t); n != 1 {
		t.Fatalf("queue holds %d entries after a failed drain, want 1", n)
	}

	if drained := sweepService(t, real).DrainImageDeletions(ctx); drained != 1 {
		t.Error("the swept upload's files were not retried")
	}
	if stillStored(t, real, img.OriginalKey) {
		t.Error("the original survived a successful retry")
	}
}

// T008 — FR-008, SC-001: one sweep removes at most its bound, oldest first.
func TestOneSweepIsBoundedAndTakesTheOldestFirst(t *testing.T) {
	_, _ = freshStore(t)
	ctx := context.Background()
	svc := sweepService(t, newImageStore(t))

	const total = 25 // more than one batch of 20
	ids := make([]uuid.UUID, total)
	for i := range ids {
		ids[i] = uploadAs(t, svc, collectorA).ID
	}
	// ids[0] oldest, ids[total-1] youngest, all past the grace period.
	for i, id := range ids {
		age(t, id, grace+time.Duration(total-i)*time.Hour)
	}

	if swept := svc.SweepAbandonedUploads(ctx); swept != 20 {
		t.Fatalf("one sweep removed %d, want its bound of 20", swept)
	}
	for i, id := range ids {
		if want := i >= 20; imageRowExists(t, id) != want {
			t.Errorf("upload %d (0 = oldest): present = %v, want %v", i, !want, want)
		}
	}
	if swept := svc.SweepAbandonedUploads(ctx); swept != total-20 {
		t.Errorf("the next sweep removed %d, want the remaining %d", swept, total-20)
	}
}

// T009 — US2/AC1, AC2, FR-004, FR-005: young uploads and referenced uploads survive.
func TestSweepLeavesYoungAndReferencedUploadsAlone(t *testing.T) {
	_, _ = freshStore(t)
	ctx := context.Background()
	svc := sweepService(t, newImageStore(t))

	young := uploadAs(t, svc, collectorA)
	age(t, young.ID, grace-time.Hour)

	referenced := uploadAs(t, svc, collectorB)
	d := draft("sweep-referenced", "Mecha Titan")
	d.ImageID = strPtr(referenced.ID.String())
	if _, v, err := svc.Add(ctx, collectorB, d); err != nil || len(v) > 0 {
		t.Fatalf("add: %v %+v", err, v)
	}
	age(t, referenced.ID, 400*24*time.Hour)

	if swept := svc.SweepAbandonedUploads(ctx); swept != 0 {
		t.Errorf("swept %d uploads, want 0", swept)
	}
	if !imageRowExists(t, young.ID) {
		t.Error("an upload younger than the grace period was swept (FR-005)")
	}
	if !imageRowExists(t, referenced.ID) {
		t.Error("an upload a collectible references was swept (FR-004)")
	}
}

// T010 — FR-006, research.md Decision 2: a save that is attaching an upload right now holds a
// key-share lock on it. The sweep must pass over that upload rather than wait for the save: it runs
// inside an upload request, and one collector's upload must not stall behind another's save (SC-005).
//
// Deterministic: the save's transaction is held open while the sweep runs, and committed after.
// Asserted first is the outcome — the saved collectible keeps its photograph — and then that the
// sweep did not wait. A plain DELETE ... WHERE NOT EXISTS, swapped in, kept the photograph but failed
// on the wait (research.md Decision 2).
func TestSweepPassesOverAnUploadASaveIsAttaching(t *testing.T) {
	_, _ = freshStore(t)
	ctx := context.Background()
	svc := sweepService(t, newImageStore(t))

	img := uploadAs(t, svc, collectorA)
	age(t, img.ID, grace+time.Hour)

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var collectibleID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO collectibles (collector_id, name, collection_status, image_id)
		VALUES ($1, 'Kaiju Sentinel', 'owned', $2) RETURNING id`,
		collectorA, img.ID).Scan(&collectibleID); err != nil {
		t.Fatalf("insert in an open transaction: %v", err)
	}

	swept := make(chan int, 1)
	go func() { swept <- svc.SweepAbandonedUploads(ctx) }()

	var finishedBeforeCommit bool
	select {
	case <-swept:
		finishedBeforeCommit = true
	case <-time.After(500 * time.Millisecond):
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit the save: %v", err)
	}
	if !finishedBeforeCommit {
		<-swept
	}

	var stored *uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT image_id FROM collectibles WHERE id = $1`,
		collectibleID).Scan(&stored); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if stored == nil || *stored != img.ID {
		t.Fatalf("the saved collectible's photograph is %v, want %v — the sweep took it (FR-006)", stored, img.ID)
	}
	if !imageRowExists(t, img.ID) {
		t.Fatal("the upload was swept although a collectible references it")
	}
	if !finishedBeforeCommit {
		t.Error("the sweep blocked on the save instead of skipping the locked upload")
	}
}

// T010 — US2/AC3, FR-006, SC-002: adds and sweeps of the same eligible upload, started together,
// many times. Each ends in one of the two permitted states; never a saved collectible whose
// photograph quietly became null.
func TestAnAddRacingTheSweepNeverLosesItsPhotograph(t *testing.T) {
	_, _ = freshStore(t)
	ctx := context.Background()
	svc := sweepService(t, newImageStore(t))

	kept, refused := 0, 0
	for i := 0; i < 40; i++ {
		img := uploadAs(t, svc, collectorA)
		age(t, img.ID, grace+time.Hour)

		d := draft(fmt.Sprintf("race-add-%d", i), "Kaiju Sentinel")
		d.ImageID = strPtr(img.ID.String())

		var (
			wg     sync.WaitGroup
			start  = make(chan struct{})
			result collection.AddResult
			v      []collectibleViolation
			addErr error
		)
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			r, violations, err := svc.Add(ctx, collectorA, d)
			result, addErr = r, err
			for _, x := range violations {
				v = append(v, collectibleViolation{x.Field, x.Message})
			}
		}()
		go func() {
			defer wg.Done()
			<-start
			time.Sleep(stagger(i))
			svc.SweepAbandonedUploads(ctx)
		}()
		close(start)
		wg.Wait()

		if addErr != nil {
			t.Fatalf("iteration %d: add failed outright: %v", i, addErr)
		}
		switch {
		case len(v) == 0:
			kept++
			var stored *uuid.UUID
			if err := pool.QueryRow(ctx, `SELECT image_id FROM collectibles WHERE id = $1`,
				result.Row.Collectible.ID).Scan(&stored); err != nil {
				t.Fatalf("iteration %d: read back: %v", i, err)
			}
			if stored == nil || *stored != img.ID {
				t.Fatalf("iteration %d: saved, but the photograph is %v — the sweep took it (FR-006)", i, stored)
			}
			if !imageRowExists(t, img.ID) {
				t.Fatalf("iteration %d: saved, but the upload is gone", i)
			}
		case len(v) == 1 && v[0].field == "imageId":
			refused++
			if imageRowExists(t, img.ID) {
				t.Fatalf("iteration %d: refused although the upload still exists", i)
			}
			if collectibleWithKey(t, d.SubmissionKey) {
				t.Fatalf("iteration %d: refused, yet a collectible was created", i)
			}
		default:
			t.Fatalf("iteration %d: unexpected violations %+v", i, v)
		}
	}
	t.Logf("40 races: %d saved with their photograph, %d refused", kept, refused)
	// Both outcomes must actually occur, or the test only ever exercised one ordering and proves
	// nothing about the other.
	if kept == 0 || refused == 0 {
		t.Errorf("only one ordering was exercised (%d saved, %d refused); adjust stagger", kept, refused)
	}
}

// stagger delays the sweep by a different amount each iteration, so across a run it lands before,
// during, and after the save — rather than winning every time because it has less work to do.
func stagger(i int) time.Duration {
	return time.Duration(i%10) * 400 * time.Microsecond
}

// T011 — FR-006: the same race, with an edit attaching the eligible upload to an existing
// collectible.
func TestAnEditRacingTheSweepNeverLosesItsPhotograph(t *testing.T) {
	_, _ = freshStore(t)
	ctx := context.Background()
	svc := sweepService(t, newImageStore(t))

	added, v, err := svc.Add(ctx, collectorA, draft("race-edit", "Mecha Titan"))
	if err != nil || len(v) > 0 {
		t.Fatalf("add: %v %+v", err, v)
	}
	current := added.Row.Collectible

	for i := 0; i < 40; i++ {
		img := uploadAs(t, svc, collectorA)
		age(t, img.ID, grace+time.Hour)

		e := edit(current, current.Version)
		e.Submitted.ImageID = strPtr(img.ID.String())

		var (
			wg         sync.WaitGroup
			start      = make(chan struct{})
			updated    postgres.Row
			violations int
			field      string
			editErr    error
		)
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			r, vs, err := svc.Edit(ctx, collectorA, current.ID, e)
			updated, editErr, violations = r, err, len(vs)
			if len(vs) > 0 {
				field = vs[0].Field
			}
		}()
		go func() {
			defer wg.Done()
			<-start
			time.Sleep(stagger(i))
			svc.SweepAbandonedUploads(ctx)
		}()
		close(start)
		wg.Wait()

		if editErr != nil {
			t.Fatalf("iteration %d: edit failed outright: %v", i, editErr)
		}
		row, err := svc.Get(ctx, collectorA, current.ID)
		if err != nil {
			t.Fatalf("iteration %d: get: %v", i, err)
		}
		switch {
		case violations == 0:
			if row.Collectible.ImageID == nil || *row.Collectible.ImageID != img.ID {
				t.Fatalf("iteration %d: edited, but the photograph is %v (FR-006)", i, row.Collectible.ImageID)
			}
			current = updated.Collectible
		case violations == 1 && field == "imageId":
			if imageRowExists(t, img.ID) {
				t.Fatalf("iteration %d: refused although the upload still exists", i)
			}
			if row.Collectible.Version != current.Version {
				t.Fatalf("iteration %d: a refused edit changed the collectible", i)
			}
		default:
			t.Fatalf("iteration %d: unexpected outcome: %d violations on %q", i, violations, field)
		}
	}
}

// T013 — US3/AC1, FR-007: saving with an upload that has been swept is refused, on the photograph,
// with a message that says to upload it again — and nothing is created.
func TestSavingWithASweptUploadIsRefusedClearly(t *testing.T) {
	_, _ = freshStore(t)
	ctx := context.Background()
	svc := sweepService(t, newImageStore(t))

	img := uploadAs(t, svc, collectorA)
	age(t, img.ID, grace+time.Hour)
	if swept := svc.SweepAbandonedUploads(ctx); swept != 1 {
		t.Fatalf("swept %d, want 1", swept)
	}

	d := draft("swept-then-saved", "Kaiju Sentinel")
	d.ImageID = strPtr(img.ID.String())
	_, v, err := svc.Add(ctx, collectorA, d)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if len(v) != 1 || v[0].Field != "imageId" {
		t.Fatalf("violations = %+v, want one on imageId", v)
	}
	if want := "That photograph is no longer available. Upload it again, or save without one."; v[0].Message != want {
		t.Errorf("message = %q, want %q", v[0].Message, want)
	}
	if collectibleWithKey(t, d.SubmissionKey) {
		t.Error("a refused save created a collectible")
	}
}

// T019 — FR-010: a sweep that fails never turns a successful upload into a failed one.
func TestAFailingSweepDoesNotFailTheUpload(t *testing.T) {
	_, _ = freshStore(t)
	ctx := context.Background()
	svc := sweepService(t, newImageStore(t))

	old := uploadAs(t, svc, collectorA)
	age(t, old.ID, grace+time.Hour)

	// Break the sweep for real: without its queue table it cannot commit.
	if _, err := pool.Exec(ctx,
		`ALTER TABLE pending_image_deletions RENAME TO pending_image_deletions_hidden`); err != nil {
		t.Fatalf("hide the queue: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`ALTER TABLE pending_image_deletions_hidden RENAME TO pending_image_deletions`); err != nil {
			t.Fatalf("restore the queue: %v", err)
		}
	})

	img, err := svc.UploadImage(ctx, collectorA, bytes.NewReader(testJPEG(t, 300, 400)))
	if err != nil {
		t.Fatalf("the upload failed because the sweep did: %v", err)
	}
	if !imageRowExists(t, img.ID) {
		t.Error("the upload reported success but has no row")
	}
	if !imageRowExists(t, old.ID) {
		t.Error("the failed sweep removed a row without queueing its files — its transaction must roll back")
	}
}

type collectibleViolation struct{ field, message string }

func collectibleWithKey(t *testing.T, key string) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(), `
		SELECT EXISTS (SELECT 1 FROM collectible_submissions WHERE submission_key = $1)`,
		key).Scan(&exists); err != nil {
		t.Fatalf("check submission: %v", err)
	}
	return exists
}
