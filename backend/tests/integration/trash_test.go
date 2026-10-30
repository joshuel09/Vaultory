//go:build integration

package integration

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/collection"
	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
	"github.com/joshuel09/vaultory/backend/internal/imagestore"
	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
)

// Feature 010: deleting moves a collectible to the Trash for 30 days, from which it can be restored
// exactly as it was, or destroyed for good.

// addWith adds a collectible with a series, a price and, optionally, a photograph.
func addWith(t *testing.T, svc *collection.Service, collector uuid.UUID, key, name string, photo bool) postgres.Row {
	t.Helper()
	ctx := context.Background()
	d := draft(key, name)
	d.Series = strPtr("Kaiju Wars")
	d.Manufacturer = strPtr("Prime 1")
	d.PurchasePrice = strPtr("1250.00")
	if photo {
		img, err := svc.UploadImage(ctx, collector, bytes.NewReader(testJPEG(t, 300, 400)))
		if err != nil {
			t.Fatalf("upload: %v", err)
		}
		d.ImageID = strPtr(img.ID.String())
	}
	added, v, err := svc.Add(ctx, collector, d)
	if err != nil || len(v) > 0 {
		t.Fatalf("add %s: %v %+v", name, err, v)
	}
	return added.Row
}

// backdateDeletion makes a trashed collectible look as if it was deleted that long ago.
func backdateDeletion(t *testing.T, id uuid.UUID, by time.Duration) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`UPDATE collectibles SET deleted_at = now() - make_interval(secs => $2) WHERE id = $1`,
		id, by.Seconds()); err != nil {
		t.Fatalf("backdate deletion: %v", err)
	}
}

func listAll(t *testing.T, svc *collection.Service, collector uuid.UUID, filter postgres.ListFilter) postgres.Page {
	t.Helper()
	page, err := svc.List(context.Background(), collector, filter, "", collection.MaxPageLimit)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	return page
}

func containsID(rows []postgres.Row, id uuid.UUID) bool {
	for _, r := range rows {
		if r.Collectible.ID == id {
			return true
		}
	}
	return false
}

func trashIDs(t *testing.T, svc *collection.Service, collector uuid.UUID) []uuid.UUID {
	t.Helper()
	page, err := svc.ListTrash(context.Background(), collector, "", collection.MaxPageLimit)
	if err != nil {
		t.Fatalf("list trash: %v", err)
	}
	ids := make([]uuid.UUID, 0, len(page.Items))
	for _, it := range page.Items {
		ids = append(ids, it.Collectible.ID)
	}
	return ids
}

func collectibleRowExists(t *testing.T, id uuid.UUID) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(),
		`SELECT EXISTS (SELECT 1 FROM collectibles WHERE id = $1)`, id).Scan(&exists); err != nil {
		t.Fatalf("check collectible: %v", err)
	}
	return exists
}

// T019 — FR-002, SC-003: a trashed collectible is in no view of the vault. Each read is exercised,
// so a future read that forgets the predicate fails here rather than in a collector's gallery.
func TestTrashIsInvisibleToEveryVaultView(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	kept := addWith(t, svc, collectorA, "trash-kept", "Mecha Titan", false)
	gone := addWith(t, svc, collectorA, "trash-gone", "Kaiju Sentinel", true)
	if err := svc.Delete(ctx, collectorA, gone.Collectible.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Get.
	if _, err := svc.Get(ctx, collectorA, gone.Collectible.ID); !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("get of a trashed collectible: err = %v, want ErrNotFound", err)
	}
	// Edit — an edit form left open across the deletion.
	if _, _, err := svc.Edit(ctx, collectorA, gone.Collectible.ID, edit(gone.Collectible, gone.Collectible.Version)); !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("edit of a trashed collectible: err = %v, want ErrNotFound", err)
	}
	// List, under every filter, and its unfiltered count.
	owned := collectible.CollectionStatus("owned")
	series, maker := "Kaiju Wars", "Prime 1"
	for name, f := range map[string]postgres.ListFilter{
		"none":         {},
		"status":       {Status: &owned},
		"series":       {Series: &series},
		"manufacturer": {Manufacturer: &maker},
	} {
		page := listAll(t, svc, collectorA, f)
		if containsID(page.Rows, gone.Collectible.ID) {
			t.Errorf("list (%s filter) includes a trashed collectible", name)
		}
		if !containsID(page.Rows, kept.Collectible.ID) {
			t.Errorf("list (%s filter) lost the collectible still in the vault", name)
		}
		if page.TotalUnfiltered != 1 {
			t.Errorf("list (%s filter): TotalUnfiltered = %d, want 1", name, page.TotalUnfiltered)
		}
	}
	// Statistics and both breakdowns.
	stats, err := svc.Statistics(ctx, collectorA)
	if err != nil {
		t.Fatalf("statistics: %v", err)
	}
	if stats.TotalCount != 1 || stats.PricedCount != 1 {
		t.Errorf("statistics count the Trash: total %d, priced %d, want 1 and 1", stats.TotalCount, stats.PricedCount)
	}
	for _, s := range stats.ByStatus {
		if s.Status == owned && s.Count != 1 {
			t.Errorf("status summary counts the Trash: owned = %d, want 1", s.Count)
		}
	}
	for name, groups := range map[string][]collectible.GroupSummary{"series": stats.BySeries, "manufacturer": stats.ByManufacturer} {
		total := 0
		for _, g := range groups {
			total += g.Count
		}
		if total != 1 {
			t.Errorf("%s breakdown counts %d collectibles, want 1", name, total)
		}
	}
}

// T020 — FR-007, FR-014, SC-002: restoring brings back every attribute, the photograph, the version
// and the gallery position; while trashed the photograph is its owner's alone.
func TestRestoreBringsBackEverything(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	older := addWith(t, svc, collectorA, "pos-older", "First", false)
	target := addWith(t, svc, collectorA, "pos-target", "Kaiju Sentinel", true)
	newer := addWith(t, svc, collectorA, "pos-newer", "Third", false)
	_ = older
	_ = newer
	before := listAll(t, svc, collectorA, postgres.ListFilter{})

	if err := svc.Delete(ctx, collectorA, target.Collectible.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	imgID := *target.Collectible.ImageID
	if rc, err := svc.OpenRendition(ctx, collectorA, imgID); err != nil {
		t.Errorf("owner cannot see a trashed collectible's photograph: %v", err)
	} else {
		_ = rc.Close()
	}
	if _, err := svc.OpenRendition(ctx, collectorB, imgID); err == nil {
		t.Error("another collector can see a trashed collectible's photograph")
	}

	restored, err := svc.Restore(ctx, collectorA, target.Collectible.ID)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	got, want := restored.Collectible, target.Collectible
	if got.Name != want.Name || *got.Series != *want.Series || got.PurchasePrice.String() != want.PurchasePrice.String() {
		t.Errorf("restored attributes differ: got %+v", got)
	}
	if got.ImageID == nil || *got.ImageID != imgID {
		t.Errorf("restored photograph = %v, want %v", got.ImageID, imgID)
	}
	if got.Version != want.Version {
		t.Errorf("restored version = %d, want %d — an edit form opened before the deletion should still save", got.Version, want.Version)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Errorf("created_at changed on restore")
	}

	after := listAll(t, svc, collectorA, postgres.ListFilter{})
	if len(after.Rows) != len(before.Rows) {
		t.Fatalf("gallery has %d entries after restore, want %d", len(after.Rows), len(before.Rows))
	}
	for i := range before.Rows {
		if before.Rows[i].Collectible.ID != after.Rows[i].Collectible.ID {
			t.Fatalf("gallery position %d changed after restore", i)
		}
	}
	if ids := trashIDs(t, svc, collectorA); len(ids) != 0 {
		t.Errorf("restored collectible is still in the Trash: %v", ids)
	}
}

// T021 — FR-004, FR-008, FR-013, SC-006.
func TestTrashRefusalsRevealNothing(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	row := addWith(t, svc, collectorA, "refuse", "Kaiju Sentinel", false)
	id := row.Collectible.ID

	// Never deleted.
	if _, err := svc.Restore(ctx, collectorA, id); !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("restore of a collectible in the vault: %v, want ErrNotFound", err)
	}

	if err := svc.Delete(ctx, collectorA, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	backdateDeletion(t, id, 5*24*time.Hour)
	// Deleting again keeps the first deletion time (FR-004).
	if err := svc.Delete(ctx, collectorA, id); err != nil {
		t.Fatalf("second delete: %v", err)
	}
	page, err := svc.ListTrash(ctx, collectorA, "", 10)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("trash: %v %+v", err, page)
	}
	if age := time.Since(page.Items[0].DeletedAt); age < 4*24*time.Hour {
		t.Errorf("deleting twice restarted the 30 days: deleted %v ago", age)
	}

	// Another collector: their Trash is empty, and their restore and permanent delete answer as for
	// an id that does not exist.
	if ids := trashIDs(t, svc, collectorB); len(ids) != 0 {
		t.Errorf("collector B sees A's Trash: %v", ids)
	}
	_, errOther := svc.Restore(ctx, collectorB, id)
	_, errAbsent := svc.Restore(ctx, collectorB, uuid.New())
	if !errors.Is(errOther, postgres.ErrNotFound) || !errors.Is(errAbsent, postgres.ErrNotFound) {
		t.Errorf("restore by another collector: %v; of an absent id: %v — both must be ErrNotFound", errOther, errAbsent)
	}
	if err := svc.DeletePermanently(ctx, collectorB, id); err != nil {
		t.Fatalf("permanent delete by another collector: %v", err)
	}
	if !collectibleRowExists(t, id) {
		t.Fatal("another collector destroyed A's trashed collectible")
	}

	// Expired: unlisted and unrestorable, even before the purge has run.
	backdateDeletion(t, id, collection.TrashRetention()+time.Minute)
	if ids := trashIDs(t, svc, collectorA); len(ids) != 0 {
		t.Errorf("an expired collectible is still listed: %v", ids)
	}
	if _, err := svc.Restore(ctx, collectorA, id); !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("restore of an expired collectible: %v, want ErrNotFound", err)
	}
}

// T022 — research.md Decision 5: a retried add after its collectible was trashed creates a new one
// and leaves the trashed one in the Trash.
func TestARetriedAddDoesNotResurrectATrashedCollectible(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	d := draft("retry-after-trash", "Kaiju Sentinel")
	first, _, err := svc.Add(ctx, collectorA, d)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := svc.Delete(ctx, collectorA, first.Row.Collectible.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	retry, v, err := svc.Add(ctx, collectorA, d)
	if err != nil || len(v) > 0 {
		t.Fatalf("retried add: %v %+v", err, v)
	}
	if retry.Row.Collectible.ID == first.Row.Collectible.ID {
		t.Fatal("the retried add returned the trashed collectible")
	}
	if ids := trashIDs(t, svc, collectorA); len(ids) != 1 || ids[0] != first.Row.Collectible.ID {
		t.Errorf("trash = %v, want just the original", ids)
	}
}

// T023 — FR-006: newest deletion first, paginated, within the window.
func TestTrashListingOrderAndPages(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	var ids []uuid.UUID
	for i := 0; i < 5; i++ {
		r := addWith(t, svc, collectorA, fmt.Sprintf("page-%d", i), fmt.Sprintf("Item %d", i), false)
		if err := svc.Delete(ctx, collectorA, r.Collectible.ID); err != nil {
			t.Fatalf("delete: %v", err)
		}
		// ids[0] deleted longest ago.
		backdateDeletion(t, r.Collectible.ID, time.Duration(5-i)*time.Hour)
		ids = append(ids, r.Collectible.ID)
	}

	var seen []uuid.UUID
	cursor := ""
	for pages := 0; ; pages++ {
		page, err := svc.ListTrash(ctx, collectorA, cursor, 2)
		if err != nil {
			t.Fatalf("list trash: %v", err)
		}
		for _, it := range page.Items {
			seen = append(seen, it.Collectible.ID)
			if !it.PurgeAt.Equal(it.DeletedAt.Add(collection.TrashRetention())) {
				t.Errorf("purgeAt %v is not deletedAt + 30 days", it.PurgeAt)
			}
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
		if pages > 5 {
			t.Fatal("pagination does not terminate")
		}
	}
	if len(seen) != 5 {
		t.Fatalf("paged through %d items, want 5", len(seen))
	}
	for i := range seen {
		if seen[i] != ids[4-i] {
			t.Fatalf("position %d is %v, want most recently deleted first", i, seen[i])
		}
	}
}

// T023 — FR-009, FR-010, SC-004: the purge destroys expired collectibles and their photographs,
// in bounded batches, and leaves the unexpired.
func TestPurgeDestroysExpiredTrashAndItsPhotographs(t *testing.T) {
	_, _ = freshStore(t)
	ctx := context.Background()
	images, err := imagestore.NewFilesystem(t.TempDir())
	if err != nil {
		t.Fatalf("image store: %v", err)
	}
	svc := collection.NewService(postgres.NewStore(pool), images, 24*time.Hour)

	fresh := addWith(t, svc, collectorA, "purge-fresh", "Recent", true)
	var expired []postgres.Row
	for i := 0; i < 22; i++ { // more than one batch of 20
		expired = append(expired, addWith(t, svc, collectorA, fmt.Sprintf("purge-%d", i), "Old", i == 0))
	}
	for _, r := range append(expired, fresh) {
		if _, err := postgres.NewStore(pool).Trash(ctx, collectorA, r.Collectible.ID); err != nil {
			t.Fatalf("trash: %v", err)
		}
	}
	for _, r := range expired {
		backdateDeletion(t, r.Collectible.ID, collection.TrashRetention()+time.Hour)
	}
	img, err := postgres.NewStore(pool).GetImage(ctx, collectorA, *expired[0].Collectible.ImageID)
	if err != nil {
		t.Fatalf("image of the first expired collectible: %v", err)
	}

	if n := svc.PurgeExpiredTrash(ctx); n != 20 {
		t.Fatalf("first purge destroyed %d, want its bound of 20", n)
	}
	if n := svc.PurgeExpiredTrash(ctx); n != 2 {
		t.Fatalf("second purge destroyed %d, want the remaining 2", n)
	}
	for _, r := range expired {
		if collectibleRowExists(t, r.Collectible.ID) {
			t.Fatalf("expired collectible %v survived", r.Collectible.ID)
		}
	}
	if !collectibleRowExists(t, fresh.Collectible.ID) {
		t.Fatal("the purge destroyed a collectible still within 30 days")
	}
	if _, err := svc.OpenRendition(ctx, collectorA, img.ID); err == nil {
		t.Error("an expired collectible's photograph is still readable")
	}
	svc.DrainImageDeletions(ctx)
	if stillStored(t, images, img.OriginalKey) || stillStored(t, images, img.RenditionKey) {
		t.Error("an expired collectible's photograph files survived the drain")
	}
}

// T024 — FR-011, SC-005.
func TestDeletePermanentlyIsImmediateAndIdempotent(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	r := addWith(t, svc, collectorA, "perm", "Kaiju Sentinel", true)
	imgID := *r.Collectible.ImageID

	// From the vault, nothing happens: it must be moved to the Trash first.
	if err := svc.DeletePermanently(ctx, collectorA, r.Collectible.ID); err != nil {
		t.Fatalf("permanent delete from the vault: %v", err)
	}
	if _, err := svc.Get(ctx, collectorA, r.Collectible.ID); err != nil {
		t.Fatalf("a collectible in the vault was destroyed without passing through the Trash: %v", err)
	}

	if err := svc.Delete(ctx, collectorA, r.Collectible.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := svc.DeletePermanently(ctx, collectorA, r.Collectible.ID); err != nil {
			t.Fatalf("permanent delete %d: %v", i+1, err)
		}
	}
	if collectibleRowExists(t, r.Collectible.ID) {
		t.Fatal("the collectible survived permanent deletion")
	}
	if _, err := svc.OpenRendition(ctx, collectorA, imgID); err == nil {
		t.Error("the photograph is still readable after permanent deletion")
	}
	if _, err := svc.Restore(ctx, collectorA, r.Collectible.ID); !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("restore after permanent deletion: %v, want ErrNotFound", err)
	}
}

// T025 — FR-012: a restore racing the purge of the same collectible ends one way or the other,
// never both. Staggered so both outcomes occur; the test fails if only one does.
func TestARestoreRacingThePurgeEndsOneWay(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()
	retention := collection.TrashRetention()

	restored, destroyed := 0, 0
	for i := 0; i < 40; i++ {
		r := addWith(t, svc, collectorA, fmt.Sprintf("race-%d", i), "Kaiju Sentinel", true)
		if err := svc.Delete(ctx, collectorA, r.Collectible.ID); err != nil {
			t.Fatalf("delete: %v", err)
		}
		// One minute inside the restore window, so a restore is still allowed — while the purge
		// below runs with a window two minutes shorter, so to it this collectible has expired. That
		// is the boundary where both can act on the same row.
		backdateDeletion(t, r.Collectible.ID, retention-time.Minute)

		var (
			wg         sync.WaitGroup
			start      = make(chan struct{})
			restoreErr error
			purged     []postgres.Purged
			purgeErr   error
		)
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, restoreErr = svc.Restore(ctx, collectorA, r.Collectible.ID)
		}()
		go func() {
			defer wg.Done()
			<-start
			time.Sleep(time.Duration(i%10) * 300 * time.Microsecond)
			purged, purgeErr = postgres.NewStore(pool).PurgeExpiredTrash(ctx, retention-2*time.Minute, 20)
		}()
		close(start)
		wg.Wait()
		if purgeErr != nil {
			t.Fatalf("iteration %d: purge: %v", i, purgeErr)
		}

		present := collectibleRowExists(t, r.Collectible.ID)
		switch {
		case restoreErr == nil && present && len(purged) == 0:
			restored++
			if _, err := svc.Get(ctx, collectorA, r.Collectible.ID); err != nil {
				t.Fatalf("iteration %d: restore reported success but the collectible is not in the vault: %v", i, err)
			}
			// Leave the vault tidy for the next iteration's counts.
			if err := svc.Delete(ctx, collectorA, r.Collectible.ID); err != nil {
				t.Fatalf("cleanup: %v", err)
			}
			if err := svc.DeletePermanently(ctx, collectorA, r.Collectible.ID); err != nil {
				t.Fatalf("cleanup: %v", err)
			}
		case errors.Is(restoreErr, postgres.ErrNotFound) && !present && len(purged) == 1:
			destroyed++
		default:
			t.Fatalf("iteration %d: restore err %v, present %v, purged %d — not one of the two outcomes",
				i, restoreErr, present, len(purged))
		}
	}
	t.Logf("40 races: %d restored, %d destroyed", restored, destroyed)
	if restored == 0 || destroyed == 0 {
		t.Errorf("only one ordering was exercised (%d restored, %d destroyed)", restored, destroyed)
	}
}

// T031 — FR-018: a photograph belongs to one collectible. Naming a trashed collectible's photograph
// is refused, so destroying it can never take another collectible's photograph with it.
func TestAPhotographCannotBeSharedWithATrashedCollectible(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	trashed := addWith(t, svc, collectorA, "shared-trashed", "Kaiju Sentinel", true)
	imgID := *trashed.Collectible.ImageID
	if err := svc.Delete(ctx, collectorA, trashed.Collectible.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	d := draft("shared-new", "Mecha Titan")
	d.ImageID = strPtr(imgID.String())
	_, v, err := svc.Add(ctx, collectorA, d)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if len(v) != 1 || v[0].Field != "imageId" {
		t.Fatalf("violations = %+v, want one on imageId", v)
	}

	// The same through an edit.
	other := addWith(t, svc, collectorA, "shared-edit", "Third", false)
	e := edit(other.Collectible, other.Collectible.Version)
	e.Submitted.ImageID = strPtr(imgID.String())
	if _, v, err := svc.Edit(ctx, collectorA, other.Collectible.ID, e); err != nil || len(v) != 1 || v[0].Field != "imageId" {
		t.Fatalf("edit naming a trashed photograph: err %v, violations %+v", err, v)
	}

	// And the photograph is still the trashed one's, to restore.
	restored, err := svc.Restore(ctx, collectorA, trashed.Collectible.ID)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if restored.Collectible.ImageID == nil || *restored.Collectible.ImageID != imgID {
		t.Errorf("restored photograph = %v, want %v", restored.Collectible.ImageID, imgID)
	}
}
