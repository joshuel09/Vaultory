package collection

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
)

// trashRetention is how long a deleted collectible is kept in the Trash before it is destroyed
// (feature 010, FR-009).
//
// One value, used for listing, for restoring, and for purging, so nothing is ever listed but
// unrestorable or restorable but unlisted (research.md Decision 3).
const trashRetention = 30 * 24 * time.Hour

// TrashRetention is trashRetention, for callers that state it — the API's purgeAt, and tests.
func TrashRetention() time.Duration { return trashRetention }

// purgeBatch bounds one purge, for the reason drainBatch bounds a drain: it runs inside a
// collector's deletion, which must not pay for everyone's expired Trash (FR-010).
const purgeBatch = 20

// TrashedCollectible is one entry in the Trash, with when it will be destroyed.
type TrashedCollectible struct {
	postgres.TrashedRow
	PurgeAt time.Time
}

// TrashPage is one page of the Trash.
type TrashPage struct {
	Items      []TrashedCollectible
	NextCursor string
}

// ListTrash returns one page of the collector's own Trash, most recently deleted first (FR-006).
func (s *Service) ListTrash(
	ctx context.Context, collectorID uuid.UUID, cursor string, limit int,
) (TrashPage, error) {
	switch {
	case limit <= 0:
		limit = DefaultPageLimit
	case limit > MaxPageLimit:
		limit = MaxPageLimit
	}
	page, err := s.store.ListTrash(ctx, collectorID, trashRetention, cursor, limit)
	if err != nil {
		return TrashPage{}, err
	}
	out := TrashPage{NextCursor: page.NextCursor, Items: make([]TrashedCollectible, 0, len(page.Rows))}
	for _, r := range page.Rows {
		// Computed here, not in the browser: when something is destroyed is a rule of the system
		// (Principle II).
		out.Items = append(out.Items, TrashedCollectible{TrashedRow: r, PurgeAt: r.DeletedAt.Add(trashRetention)})
	}
	return out, nil
}

// Restore returns a collectible from the Trash to the vault, exactly as it was (FR-007).
//
// postgres.ErrNotFound when there is nothing of this collector's to restore, for any reason (FR-008).
// Unlike deletion, that is an error the collector sees: they need to know it did not come back.
func (s *Service) Restore(ctx context.Context, collectorID, collectibleID uuid.UUID) (postgres.Row, error) {
	row, err := s.store.Restore(ctx, collectorID, collectibleID, trashRetention)
	if err != nil {
		return postgres.Row{}, err
	}
	slog.InfoContext(ctx, "collectible restored from trash",
		"collector_id", collectorID.String(),
		"collectible_id", collectibleID.String(),
	)
	return row, nil
}

// DeletePermanently destroys a collectible in the Trash, and its photograph, now (FR-011).
//
// Success whether or not there was anything to destroy, as for any deletion (006 FR-025).
func (s *Service) DeletePermanently(ctx context.Context, collectorID, collectibleID uuid.UUID) error {
	destroyed, err := s.store.DeletePermanently(ctx, collectorID, collectibleID)
	if err != nil {
		return err
	}
	if destroyed {
		slog.InfoContext(ctx, "collectible deleted permanently",
			"collector_id", collectorID.String(),
			"collectible_id", collectibleID.String(),
		)
	}
	s.DrainImageDeletions(ctx)
	return nil
}

// PurgeExpiredTrash destroys one batch of collectibles that have outlived trashRetention (FR-010).
//
// Runs after each deletion and once at start-up — no ticker, for the reason DrainImageDeletions has
// none. Allowed to fail: nothing a collector is doing depends on it, so an error is recorded and the
// next deletion tries again. Callers drain afterwards to remove the destroyed photographs' files.
//
// Returns how many it destroyed.
func (s *Service) PurgeExpiredTrash(ctx context.Context) int {
	purged, err := s.store.PurgeExpiredTrash(ctx, trashRetention, purgeBatch)
	if err != nil {
		slog.ErrorContext(ctx, "could not purge expired trash", "error", err)
		return 0
	}
	for _, p := range purged {
		// Identifiers only, one record per collectible, as for every other way a collectible
		// leaves (FR-015).
		slog.InfoContext(ctx, "collectible expired from trash",
			"collector_id", p.CollectorID.String(),
			"collectible_id", p.CollectibleID.String(),
		)
	}
	return len(purged)
}
