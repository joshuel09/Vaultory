package collection

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// Delete moves a collectible to the Trash, where it is kept for trashRetention and can be restored
// (006 FR-022, as amended by feature 010 FR-001).
//
// Deleting something that is not in this collector's vault is a success, not an error — already
// deleted, never existed, or another collector's, the end state the collector asked for already
// holds (006 FR-025). That rule lives here rather than in the browser, because "deleting twice is
// not an error" is a rule of the system and Principle II keeps business rules out of the frontend.
func (s *Service) Delete(ctx context.Context, collectorID, collectibleID uuid.UUID) error {
	moved, err := s.store.Trash(ctx, collectorID, collectibleID)
	if err != nil {
		return err
	}

	if moved {
		/*
		 * The operational record of a deletion (006 FR-040, FR-041; feature 010 FR-015).
		 *
		 * Identifiers only. Never the name, the notes, or any other attribute: a record that
		 * echoes what a collector typed is a record that leaks their collection into the logs.
		 *
		 * Only when something actually moved. An idempotent no-op is not a deletion, and recording
		 * one would make the record wrong precisely when somebody is using it to work out where a
		 * collectible went.
		 */
		slog.InfoContext(ctx, "collectible moved to trash",
			"collector_id", collectorID.String(),
			"collectible_id", collectibleID.String(),
		)
	}

	// Deletions are what fill the Trash, so they are what empty it of anything past retention —
	// bounded, and unable to fail this deletion (FR-010). Then the files of whatever that destroyed.
	s.PurgeExpiredTrash(ctx)
	s.DrainImageDeletions(ctx)
	// And audit entries past their year, for the same reason the drain runs here: without a
	// schedule, the work that creates a backlog is what works it (feature 014, FR-007).
	s.PurgeExpiredAudit(ctx)
	return nil
}

// auditPurgeBatch bounds one retention pass, so it never makes a deletion do unbounded work.
const auditPurgeBatch = 500

// PurgeExpiredAudit removes one batch of audit entries over a year old (feature 014, FR-007). Like
// the drain, it is allowed to fail: nothing a collector does depends on it, so an error is
// recorded and nothing more.
func (s *Service) PurgeExpiredAudit(ctx context.Context) int {
	n, err := s.store.PurgeExpiredAudit(ctx, auditPurgeBatch)
	if err != nil {
		slog.ErrorContext(ctx, "could not purge expired audit entries", "error", err)
		return 0
	}
	if n > 0 {
		slog.InfoContext(ctx, "purged expired audit entries", "entries", n)
	}
	return n
}
