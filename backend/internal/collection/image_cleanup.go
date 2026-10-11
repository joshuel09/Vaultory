package collection

import (
	"context"
	"log/slog"
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
