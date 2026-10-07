//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
)

// T053 — FR-024, FR-026: deleting removes exactly the one entry and nothing else.
func TestDeleteRemovesExactlyOneCollectible(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	// Two identical entries, which is the situation deleting exists for: feature 001 shipped a
	// submission-key defence precisely because a duplicate would otherwise be permanent.
	first, _, err := svc.Add(ctx, collectorA, draft("del-1", "Kaiju Sentinel"))
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	twin, _, err := svc.Add(ctx, collectorA, draft("del-2", "Kaiju Sentinel"))
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	other := draft("del-3", "Harbour Golem")
	other.Status = "sold"
	if _, _, err := svc.Add(ctx, collectorA, other); err != nil {
		t.Fatalf("add: %v", err)
	}

	if err := svc.Delete(ctx, collectorA, first.Row.Collectible.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Gone from the gallery.
	page, err := svc.List(ctx, collectorA, nil, "", 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Rows) != 2 {
		t.Fatalf("the collection holds %d entries after one deletion, want 2", len(page.Rows))
	}
	for _, r := range page.Rows {
		if r.Collectible.ID == first.Row.Collectible.ID {
			t.Error("the deleted collectible is still in the gallery (FR-026)")
		}
	}

	// The identical twin is untouched, which is the whole point of deleting one duplicate.
	if _, err := svc.Get(ctx, collectorA, twin.Row.Collectible.ID); err != nil {
		t.Errorf("the twin went with it: %v — deleting reaches one entry only (FR-024)", err)
	}

	// And gone from every status filter, not merely from the unfiltered view.
	sold, _ := collectible.ParseCollectionStatus("sold")
	filtered, err := svc.List(ctx, collectorA, &sold, "", 50)
	if err != nil {
		t.Fatalf("filtered list: %v", err)
	}
	if len(filtered.Rows) != 1 {
		t.Errorf("the sold filter holds %d entries, want 1", len(filtered.Rows))
	}
	if filtered.TotalUnfiltered != 2 {
		t.Errorf("totalUnfiltered = %d after a deletion, want 2 — the count the empty state and "+
			"the no-results state are told apart by", filtered.TotalUnfiltered)
	}
}

// T053 — FR-025: deleting something that is not in this collector's vault is a success.
func TestDeletingIsIdempotentAndRevealsNothing(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	added, _, err := svc.Add(ctx, collectorA, draft("del-idem", "Kaiju Sentinel"))
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	id := added.Row.Collectible.ID

	if err := svc.Delete(ctx, collectorA, id); err != nil {
		t.Fatalf("first delete: %v", err)
	}
	// A retry after a lost response, or a second click. The end state the collector asked for
	// already holds, so this is not an error.
	if err := svc.Delete(ctx, collectorA, id); err != nil {
		t.Errorf("deleting an already-deleted collectible returned %v, want success (FR-025)", err)
	}
	if err := svc.Delete(ctx, collectorA, uuid.New()); err != nil {
		t.Errorf("deleting a collectible that never existed returned %v, want success", err)
	}
}

// T053 — FR-031: another collector's collectible is not deleted, and the answer is the same one a
// fictional identifier gets.
func TestDeletingAnotherCollectorsCollectible(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	added, _, err := svc.Add(ctx, collectorA, draft("del-theirs", "Private Statue"))
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	mine := added.Row.Collectible.ID

	// Succeeds, and that is correct: the state collector B asked for — that collectible not being
	// in their vault — already holds. Answering anything else would tell B the id is real.
	if err := svc.Delete(ctx, collectorB, mine); err != nil {
		t.Errorf("B's delete of A's collectible returned %v; it must be indistinguishable from "+
			"deleting a fictional id (FR-031)", err)
	}

	if _, err := svc.Get(ctx, collectorA, mine); err != nil {
		t.Fatalf("collector A's collectible was actually deleted by collector B: %v", err)
	}
}

// T054 — a collectible added inside the idempotency window deletes cleanly, and replaying that add
// afterwards creates a new one rather than failing.
func TestDeletingACollectibleAddedWithinTheIdempotencyWindow(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	added, _, err := svc.Add(ctx, collectorA, draft("del-window", "Kaiju Sentinel"))
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	// The submission row references the collectible; without ON DELETE CASCADE this fails.
	if err := svc.Delete(ctx, collectorA, added.Row.Collectible.ID); err != nil {
		t.Fatalf("deleting a collectible with a live submission key: %v", err)
	}

	// Replaying the same key now creates a new collectible. The spec accepts this rather than
	// resurrecting the deleted one — the deletion stands.
	replay, _, err := svc.Add(ctx, collectorA, draft("del-window", "Kaiju Sentinel"))
	if err != nil {
		t.Fatalf("replayed add after deletion: %v", err)
	}
	if replay.Existing {
		t.Error("the replay was answered with a collectible that no longer exists")
	}
	if replay.Row.Collectible.ID == added.Row.Collectible.ID {
		t.Error("the replay resurrected the deleted collectible")
	}
}

// T055 — FR-020: a deleted collectible's photograph stops being retrievable at once.
func TestDeletingTakesThePhotographWithIt(t *testing.T) {
	store, svc := freshStore(t)
	ctx := context.Background()

	img, err := svc.UploadImage(ctx, collectorA, bytes.NewReader(testJPEG(t, 300, 400)))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	d := draft("del-photo", "Kaiju Sentinel")
	d.ImageID = strPtr(img.ID.String())
	added, v, err := svc.Add(ctx, collectorA, d)
	if err != nil || len(v) > 0 {
		t.Fatalf("add: %v %+v", err, v)
	}

	// Fetchable by its owner beforehand.
	if rc, err := svc.OpenRendition(ctx, collectorA, img.ID); err != nil {
		t.Fatalf("the rendition should be readable before the deletion: %v", err)
	} else {
		_ = rc.Close()
	}

	if err := svc.Delete(ctx, collectorA, added.Row.Collectible.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Unfetchable immediately, by its owner. Deleting the image row is what achieves this —
	// unlinking would not, because a rendition is authorized on the image's own collector_id.
	if _, err := svc.OpenRendition(ctx, collectorA, img.ID); err == nil {
		t.Error("the photograph of a deleted collectible is still readable by its owner (FR-020)")
	}
	if _, err := store.GetImage(ctx, collectorA, img.ID); err == nil {
		t.Error("the collectible_images row survived the deletion")
	}
}

// T056 — FR-040, FR-041: a deletion leaves a server-side record, carrying identifiers and no
// collection content, and only when a row was actually deleted.
func TestADeletionIsRecordedWithoutCollectionContent(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	d := draft("del-log", "Kaiju Sentinel")
	d.Notes = strPtr("Box has a small dent on the lower left corner.")
	d.Series = strPtr("Kaiju Wars")
	added, _, err := svc.Add(ctx, collectorA, d)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	id := added.Row.Collectible.ID

	var captured bytes.Buffer
	restore := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&captured, &slog.HandlerOptions{Level: slog.LevelInfo})))
	defer slog.SetDefault(restore)

	if err := svc.Delete(ctx, collectorA, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	// A second delete removes nothing, so it must record nothing — a record of a deletion that
	// did not happen is wrong in exactly the situation the record exists to explain.
	if err := svc.Delete(ctx, collectorA, id); err != nil {
		t.Fatalf("second delete: %v", err)
	}

	var records []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(captured.String()), "\n") {
		if line == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("log line is not JSON: %q", line)
		}
		if msg, _ := entry["msg"].(string); strings.Contains(strings.ToLower(msg), "delet") {
			records = append(records, entry)
		}
	}

	if len(records) != 1 {
		t.Fatalf("%d deletion records for one actual deletion and one no-op, want 1 (FR-040)", len(records))
	}
	record := records[0]

	if !strings.Contains(flatten(record), id.String()) {
		t.Errorf("the record does not identify which collectible was deleted: %v", record)
	}
	if !strings.Contains(flatten(record), collectorA.String()) {
		t.Errorf("the record does not identify whose it was: %v", record)
	}
	// FR-040: identifiers only. A record that echoes a collector's notes is a record that leaks
	// their collection into the logs.
	for _, content := range []string{"Kaiju Sentinel", "Kaiju Wars", "lower left corner"} {
		if strings.Contains(flatten(record), content) {
			t.Errorf("the deletion record carried collection content %q: %v", content, record)
		}
	}
}

func flatten(m map[string]any) string {
	b, _ := json.Marshal(m)
	return string(b)
}
