package collection

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// Delete removes a collectible permanently (FR-022).
//
// There is no undo, no trash, and no restore. A collector who deletes something and wants it back
// must add it again.
//
// Deleting something that is not in this collector's vault is a success, not an error — already
// deleted, never existed, or another collector's, the end state the collector asked for already
// holds (FR-025). That rule lives here rather than in the browser, because "deleting twice is not
// an error" is a rule of the system and Principle II keeps business rules out of the frontend.
func (s *Service) Delete(ctx context.Context, collectorID, collectibleID uuid.UUID) error {
	deleted, err := s.store.Delete(ctx, collectorID, collectibleID)
	if err != nil {
		return err
	}

	if deleted {
		/*
		 * The only trace a deletion leaves (FR-040, FR-041).
		 *
		 * A log rather than a table, deliberately. There is no query path from the application to
		 * it, so it cannot be surfaced in the product by accident and cannot quietly become the
		 * trash bin this feature does not build — a table called deleted_collectibles would be one
		 * pull request away from a restore feature.
		 *
		 * Identifiers only. Never the name, the notes, or any other attribute: a record that
		 * echoes what a collector typed is a record that leaks their collection into the logs.
		 *
		 * Only when a row was actually removed. An idempotent no-op is not a deletion, and
		 * recording one would make the record wrong precisely when somebody is using it to work
		 * out where a collectible went.
		 */
		slog.InfoContext(ctx, "collectible deleted",
			"collector_id", collectorID.String(),
			"collectible_id", collectibleID.String(),
		)
	}

	// Whatever photograph that deletion orphaned. After the commit, and unable to fail it: the
	// image is already unreachable, so this is reclaiming bytes rather than keeping a promise
	// (FR-020a).
	s.DrainImageDeletions(ctx)
	return nil
}
