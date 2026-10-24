package collection

import (
	"context"
	"log/slog"
	"time"
)

// drainBatch is how many queued image deletions one drain attempts.
//
// Bounded because draining happens inside a collector's edit or delete request. An unbounded drain
// would make one person's deletion pay for every file anybody has ever orphaned, and the work is
// never urgent — the images are already unreachable.
const drainBatch = 20

// DrainImageDeletions removes the files of images whose database rows are already gone.
//
// Nothing a collector can see depends on this succeeding. Deleting the collectible_images row is
// what makes an image unfetchable, and that happens transactionally with the edit or deletion that
// orphaned it (research.md Decision 4). This is hygiene: it reclaims the bytes.
//
// That is precisely why it is allowed to fail. A file that cannot be deleted keeps its queue entry
// and is attempted again on the next edit or deletion, and at the next start-up — retried rather
// than abandoned (FR-020a). Making a collector's deletion wait on the filesystem, or fail with it,
// would be the worse bargain: a storage fault would make collectibles undeletable.
//
// Returns the number of images whose files were removed.
func (s *Service) DrainImageDeletions(ctx context.Context) int {
	pending, err := s.store.PendingImageDeletions(ctx, drainBatch)
	if err != nil {
		slog.ErrorContext(ctx, "could not read pending image deletions", "error", err)
		return 0
	}

	drained := 0
	for _, p := range pending {
		// imagestore.Delete treats an absent object as success, so a half-finished earlier attempt
		// does not wedge the queue.
		if err := s.images.Delete(ctx, p.OriginalKey); err != nil {
			slog.WarnContext(ctx, "could not delete original image file; will retry",
				"image_id", p.ImageID, "error", err)
			continue
		}
		if err := s.images.Delete(ctx, p.RenditionKey); err != nil {
			slog.WarnContext(ctx, "could not delete image rendition; will retry",
				"image_id", p.ImageID, "error", err)
			continue
		}
		// Only now. Forgetting the entry before the files are gone would turn a storage failure
		// into a permanent leak.
		if err := s.store.ForgetImageDeletion(ctx, p.ImageID); err != nil {
			slog.ErrorContext(ctx, "deleted image files but could not clear the queue entry",
				"image_id", p.ImageID, "error", err)
			continue
		}
		drained++
	}
	return drained
}

// abandonedUploadGrace is how long an upload nobody has attached may wait before it is swept.
//
// Thirty days: the time a session survives unused, so an upload older than this belongs, in the
// ordinary case, to a form that has outlived the session it was made in. It is not a guarantee —
// signing in again in another tab revives an old form — and nothing relies on it being one. The
// guarantee is the locking in SweepAbandonedImages, and a form past this age gets a refusal that
// says to upload the photograph again (feature 009, FR-002, FR-006, FR-007).
const abandonedUploadGrace = 30 * 24 * time.Hour

// sweepBatch bounds one sweep, for the same reason drainBatch bounds a drain: it runs inside a
// collector's upload, which must not pay for the whole backlog (FR-008).
const sweepBatch = 20

// SweepAbandonedUploads removes one batch of uploads nobody attached within the grace period. Their
// files go on the deletion queue; the caller drains it.
//
// It runs after an upload and once at start-up — never on a ticker, for the reason
// DrainImageDeletions has none: abandoned uploads accrue only as fast as uploads happen, so sweeping
// on upload works the backlog exactly as it grows (research.md Decision 3 of feature 009).
//
// Like the drain, it is allowed to fail. Nothing a collector does depends on it, so an error is
// recorded and nothing more (FR-010). Returns how many uploads it removed.
func (s *Service) SweepAbandonedUploads(ctx context.Context) int {
	swept, err := s.store.SweepAbandonedImages(ctx, abandonedUploadGrace, sweepBatch)
	if err != nil {
		slog.ErrorContext(ctx, "could not sweep abandoned uploads", "error", err)
		return 0
	}
	if swept > 0 {
		slog.InfoContext(ctx, "swept abandoned uploads", "images", swept)
	}
	return swept
}
